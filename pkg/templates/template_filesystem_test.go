package templates_test

// cSpell: words filesys gotmpl sprig tmpl myresource missingkey testdir
import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/templates"
)

func TestNewTemplateFS(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	data := map[string]any{"key": "value"}
	tfs := templates.NewTemplateFS(fs, data)
	req.NotNil(tfs, "TemplateFS should not be nil")
}

func TestTemplateFS_ReadFile_NonTemplate(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := []byte("plain content, no template")
	err := fs.WriteFile("plain.yaml", content)
	req.NoError(err)

	tfs := templates.NewTemplateFS(fs, map[string]any{})
	got, err := tfs.ReadFile("plain.yaml")
	req.NoError(err)
	req.Equal(content, got, "non-template file should be returned as-is")
}

func TestTemplateFS_ReadFile_TmplFile(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := []byte("name: {{ .Values.data.name }}")
	err := fs.WriteFile("resource.yaml.tmpl", content)
	req.NoError(err)

	data := map[string]any{
		"data": map[string]any{
			"name": "my-resource",
		},
	}
	tfs := templates.NewTemplateFS(fs, data)
	got, err := tfs.ReadFile("resource.yaml.tmpl")
	req.NoError(err)
	req.Equal("name: my-resource", string(got), "template should be rendered with data")
}

func TestTemplateFS_ReadFile_GotmplFile(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := []byte("repoURL: git@github.com:{{ .Values.data.org }}/{{ .Values.data.repo }}.git")
	err := fs.WriteFile("application.yaml.gotmpl", content)
	req.NoError(err)

	data := map[string]any{
		"data": map[string]any{
			"org":  "my-org",
			"repo": "my-repo",
		},
	}
	tfs := templates.NewTemplateFS(fs, data)
	got, err := tfs.ReadFile("application.yaml.gotmpl")
	req.NoError(err)
	req.Equal("repoURL: git@github.com:my-org/my-repo.git", string(got))
}

func TestTemplateFS_ReadFile_SprigFunctions(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	// Use a sprig function (upper) to verify sprig funcs are available
	content := []byte("name: {{ .Values.data.name | upper }}")
	err := fs.WriteFile("resource.yaml.tmpl", content)
	req.NoError(err)

	data := map[string]any{
		"data": map[string]any{
			"name": "myresource",
		},
	}
	tfs := templates.NewTemplateFS(fs, data)
	got, err := tfs.ReadFile("resource.yaml.tmpl")
	req.NoError(err)
	req.Equal("name: MYRESOURCE", string(got))
}

func TestTemplateFS_ReadFile_MissingKey(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	// missingkey=error option causes error for missing keys
	content := []byte("name: {{ .Values.data.missing_key }}")
	err := fs.WriteFile("resource.yaml.tmpl", content)
	req.NoError(err)

	data := map[string]any{
		"data": map[string]any{},
	}
	tfs := templates.NewTemplateFS(fs, data)
	_, err = tfs.ReadFile("resource.yaml.tmpl")
	req.Error(err, "should error on missing template key")
	req.Contains(err.Error(), "resource.yaml.tmpl")
}

func TestTemplateFS_ReadFile_FileNotFound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	tfs := templates.NewTemplateFS(fs, map[string]any{})
	_, err := tfs.ReadFile("nonexistent.yaml")
	req.Error(err, "should error when file not found")
}

func TestTemplateFS_PassthroughMethods(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	tfs := templates.NewTemplateFS(fs, map[string]any{})

	// Test Mkdir and IsDir
	err := tfs.Mkdir("testdir")
	req.NoError(err)
	req.True(tfs.IsDir("testdir"), "newly created directory should exist")

	// Test Exists
	err = tfs.WriteFile("testdir/file.txt", []byte("content"))
	req.NoError(err)
	req.True(tfs.Exists("testdir/file.txt"), "file should exist after writing")
	req.False(tfs.Exists("nonexistent.txt"), "nonexistent file should not exist")

	// Test ReadDir
	entries, err := tfs.ReadDir("testdir")
	req.NoError(err)
	req.Contains(entries, "file.txt")

	// Test MkdirAll
	err = tfs.MkdirAll("a/b/c")
	req.NoError(err)
	req.True(tfs.IsDir("a/b/c"))

	// Test RemoveAll
	err = tfs.RemoveAll("testdir")
	req.NoError(err)
	req.False(tfs.Exists("testdir/file.txt"), "file should not exist after RemoveAll")
}

func TestTemplateFS_WriteFile(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	tfs := templates.NewTemplateFS(fs, map[string]any{})

	err := tfs.WriteFile("output.yaml", []byte("content"))
	req.NoError(err)
	// Verify it was written to the underlying FS
	content, err := tfs.ReadFile("output.yaml")
	req.NoError(err)
	req.Equal([]byte("content"), content)
}

func TestTemplateFS_InvalidTemplate(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	// Invalid Go template syntax
	content := []byte("name: {{ .Values.data.name")
	err := fs.WriteFile("invalid.yaml.tmpl", content)
	req.NoError(err)

	tfs := templates.NewTemplateFS(fs, map[string]any{"data": map[string]any{}})
	_, err = tfs.ReadFile("invalid.yaml.tmpl")
	req.Error(err, "should error on invalid template syntax")
	req.Contains(err.Error(), "invalid.yaml.tmpl")
}
