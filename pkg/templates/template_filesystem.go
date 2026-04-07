// cSpell: words filesys tmpl gotmpl missingkey wrapcheck

//nolint:wrapcheck // No added value in wrapping errors in this package.
package templates

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"text/template"

	sprig "github.com/go-task/slim-sprig/v3"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var _ filesys.FileSystem = (*TemplateFS)(nil)

type TemplateFS struct {
	fs   filesys.FileSystem
	data any
}

func NewTemplateFS(fs filesys.FileSystem, data any) *TemplateFS {
	return &TemplateFS{fs: fs, data: data}
}

// CleanedAbs implements [filesys.FileSystem].
func (t *TemplateFS) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return t.fs.CleanedAbs(path)
}

// Create implements [filesys.FileSystem].
func (t *TemplateFS) Create(path string) (filesys.File, error) {
	return t.fs.Create(path)
}

// Exists implements [filesys.FileSystem].
func (t *TemplateFS) Exists(path string) bool {
	return t.fs.Exists(path)
}

// Glob implements [filesys.FileSystem].
func (t *TemplateFS) Glob(pattern string) ([]string, error) {
	return t.fs.Glob(pattern)
}

// IsDir implements [filesys.FileSystem].
func (t *TemplateFS) IsDir(path string) bool {
	return t.fs.IsDir(path)
}

// Mkdir implements [filesys.FileSystem].
func (t *TemplateFS) Mkdir(path string) error {
	return t.fs.Mkdir(path)
}

// MkdirAll implements [filesys.FileSystem].
func (t *TemplateFS) MkdirAll(path string) error {
	return t.fs.MkdirAll(path)
}

// Open implements [filesys.FileSystem].
func (t *TemplateFS) Open(path string) (filesys.File, error) {
	return t.fs.Open(path)
}

// ReadDir implements [filesys.FileSystem].
func (t *TemplateFS) ReadDir(path string) ([]string, error) {
	return t.fs.ReadDir(path)
}

type TemplateData struct {
	Values any
}

func renderTemplate(content []byte, name string, data *TemplateData) (string, error) {
	t, err := template.New(name).
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=error").
		Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("while parsing template %s: %w", name, err)
	}
	var output strings.Builder
	err = t.Execute(&output, data)
	if err != nil {
		slog.Debug("Error executing template", "template", name, "error", err.Error())
		return "", fmt.Errorf("while executing template %s: %w", name, err)
	}
	return output.String(), nil
}

// ReadFile implements [filesys.FileSystem].
func (t *TemplateFS) ReadFile(path string) ([]byte, error) {
	content, err := t.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// if path ends with .tmpl, render the template with the data
	ext := filepath.Ext(path)
	if ext == ".tmpl" || ext == ".gotmpl" {
		slog.Debug("Rendering template", "path", path, "length", len(content))
		rendered, err := renderTemplate(content, path, &TemplateData{Values: t.data})
		if err != nil {
			return nil, fmt.Errorf("while rendering template %s: %w", path, err)
		}
		content = []byte(rendered)
	}
	return content, nil
}

// RemoveAll implements [filesys.FileSystem].
func (t *TemplateFS) RemoveAll(path string) error {
	return t.fs.RemoveAll(path)
}

// Walk implements [filesys.FileSystem].
func (t *TemplateFS) Walk(path string, walkFn filepath.WalkFunc) error {
	return t.fs.Walk(path, walkFn)
}

// WriteFile implements [filesys.FileSystem].
func (t *TemplateFS) WriteFile(path string, data []byte) error {
	return t.fs.WriteFile(path, data)
}
