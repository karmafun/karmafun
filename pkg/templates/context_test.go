package templates_test

// cSpell: words filesys testify karmafun gotmpl
import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/templates"
)

func TestMergeMaps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    map[string]any
		b    map[string]any
		want map[string]any
	}{
		{
			name: "Empty maps return empty map",
			a:    map[string]any{},
			b:    map[string]any{},
			want: map[string]any{},
		},
		{
			name: "B values override A values",
			a:    map[string]any{"key": "valueA"},
			b:    map[string]any{"key": "valueB"},
			want: map[string]any{"key": "valueB"},
		},
		{
			name: "Nil value in B skips the key",
			a:    map[string]any{"key": "valueA"},
			b:    map[string]any{"key": nil},
			want: map[string]any{"key": "valueA"},
		},
		{
			name: "Keys only in A are preserved",
			a:    map[string]any{"onlyA": "value"},
			b:    map[string]any{},
			want: map[string]any{"onlyA": "value"},
		},
		{
			name: "Keys only in B are added",
			a:    map[string]any{},
			b:    map[string]any{"onlyB": "value"},
			want: map[string]any{"onlyB": "value"},
		},
		{
			name: "Nested maps are merged recursively",
			a: map[string]any{
				"nested": map[string]any{
					"fromA": "valueA",
					"both":  "fromA",
				},
			},
			b: map[string]any{
				"nested": map[string]any{
					"fromB": "valueB",
					"both":  "fromB",
				},
			},
			want: map[string]any{
				"nested": map[string]any{
					"fromA": "valueA",
					"fromB": "valueB",
					"both":  "fromB",
				},
			},
		},
		{
			name: "B non-map replaces A map",
			a:    map[string]any{"key": map[string]any{"nested": "value"}},
			b:    map[string]any{"key": "scalar"},
			want: map[string]any{"key": "scalar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := templates.MergeMaps(tt.a, tt.b)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewPlatformValues(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	req.NotNil(pv)
	req.Equal("config.karmafun.dev/v1alpha1", pv.APIVersion)
	req.Equal("PlatformValues", pv.Kind)
	req.Equal("platform-values", pv.Name)
	req.NotNil(pv.Data)
	req.Empty(pv.Data)
}

func TestPlatformValues_AsMap(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{
		"key": "value",
		"nested": map[string]any{
			"inner": "inner-value",
		},
	}
	m := pv.AsMap()
	req.Contains(m, "data")
	data, ok := m["data"].(map[string]any)
	req.True(ok, "data should be map[string]any")
	req.Equal("value", data["key"])
	nested, ok := data["nested"].(map[string]any)
	req.True(ok, "nested should be map[string]any")
	req.Equal("inner-value", nested["inner"])
}

func TestReadPlatformValues_FileNotFound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	pv, err := templates.ReadPlatformValues(fs, "values.yaml")
	req.NoError(err)
	req.NotNil(pv)
	req.Empty(pv.Data, "should return empty values when file not found")
}

func TestReadPlatformValues_ValidFile(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := `apiVersion: config.karmafun.dev/v1alpha1
kind: PlatformValues
metadata:
  name: test-values
data:
  domain: example.com
  project:
    name: my-project
`
	err := fs.WriteFile("values.yaml", []byte(content))
	req.NoError(err)

	pv, err := templates.ReadPlatformValues(fs, "values.yaml")
	req.NoError(err)
	req.NotNil(pv)
	req.Equal("example.com", pv.Data["domain"])
	projectData, ok := pv.Data["project"].(map[string]any)
	req.True(ok, "project should be map[string]any")
	req.Equal("my-project", projectData["name"])
}

func TestReadPlatformValues_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	err := fs.WriteFile("values.yaml", []byte("not: valid: yaml: {"))
	req.NoError(err)

	_, err = templates.ReadPlatformValues(fs, "values.yaml")
	req.Error(err)
	req.Contains(err.Error(), "values.yaml")
}

func TestMergeValues_BothNonNil(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{
		"fromPlatform": "platformValue",
		"shared":       "platformShared",
	}
	sv := templates.NewPlatformValues()
	sv.Data = map[string]any{
		"fromSecrets": "secretValue",
		"shared":      "secretShared",
	}

	merged := templates.MergeValues(pv, sv)
	req.NotNil(merged)
	req.Equal("platformValue", merged.Data["fromPlatform"])
	req.Equal("secretValue", merged.Data["fromSecrets"])
	req.Equal("secretShared", merged.Data["shared"], "secrets should override platform values")
}

func TestMergeValues_NilPlatform(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	sv := templates.NewPlatformValues()
	sv.Data = map[string]any{"key": "value"}
	merged := templates.MergeValues(nil, sv)
	req.Equal(sv, merged)
}

func TestMergeValues_NilSecrets(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{"key": "value"}
	merged := templates.MergeValues(pv, nil)
	req.Equal(pv, merged)
}
