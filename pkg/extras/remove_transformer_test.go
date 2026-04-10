package extras_test

// cSpell: words yamls

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
)

func makeResMap(t *testing.T, yamls ...string) resmap.ResMap {
	t.Helper()
	rm := resmap.New()
	for _, y := range yamls {
		node, err := yaml.Parse(y)
		if err != nil {
			t.Fatalf("parsing yaml: %v", err)
		}
		r := &resource.Resource{RNode: *node}
		if err := rm.Append(r); err != nil {
			t.Fatalf("appending resource: %v", err)
		}
	}
	return rm
}

func TestNewRemoveTransformerPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	req.NotNil(plugin)
}

func TestRemoveTransformerPlugin_Config(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	config := []byte(`
targets:
  - kind: ConfigMap
    name: test-cm
`)
	err := plugin.Config(nil, config)
	req.NoError(err)
}

func TestRemoveTransformerPlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	err := plugin.Config(nil, []byte("{invalid yaml"))
	req.Error(err)
}

func TestRemoveTransformerPlugin_Transform_NoTargets(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	err := plugin.Config(nil, []byte(`{}`))
	req.NoError(err)

	rm := makeResMap(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
`)
	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "at least one target")
}

func TestRemoveTransformerPlugin_Transform(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                   string
		config                 string
		input                  []string
		expectedRemainingNames []string
		expectedStartCount     int
		expectedEndCount       int
	}{
		{
			name: "Remove by kind and name",
			config: `
targets:
  - kind: ConfigMap
    name: to-remove
`,
			input: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: to-remove
`,
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: keep-me
`,
			},
			expectedStartCount:     2,
			expectedEndCount:       1,
			expectedRemainingNames: []string{"keep-me"},
		},
		{
			name: "Remove by kind only",
			config: `
targets:
  - kind: Secret
`,
			input: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: keep-cm
`,
				`apiVersion: v1
kind: Secret
metadata:
  name: remove-secret
`,
			},
			expectedStartCount:     2,
			expectedEndCount:       1,
			expectedRemainingNames: []string{"keep-cm"},
		},
		{
			name: "Remove multiple targets",
			config: `
targets:
  - kind: ConfigMap
    name: remove1
  - kind: ConfigMap
    name: remove2
`,
			input: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: remove1
`,
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: remove2
`,
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: keep-me
`,
			},
			expectedStartCount:     3,
			expectedEndCount:       1,
			expectedRemainingNames: []string{"keep-me"},
		},
		{
			name: "No matching resources",
			config: `
targets:
  - kind: Deployment
    name: non-existent
`,
			input: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: keep-cm
`,
				`apiVersion: v1
kind: Secret
metadata:
  name: keep-secret
`,
			},
			expectedStartCount:     2,
			expectedEndCount:       2,
			expectedRemainingNames: []string{"keep-cm", "keep-secret"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			plugin := extras.NewRemoveTransformerPlugin()
			err := plugin.Config(nil, []byte(tc.config))
			req.NoError(err)

			rm := makeResMap(t, tc.input...)
			req.Equal(tc.expectedStartCount, rm.Size())

			err = plugin.Transform(rm)
			req.NoError(err)
			req.Equal(tc.expectedEndCount, rm.Size())

			resources := rm.Resources()
			var remainingNames []string
			for _, r := range resources {
				remainingNames = append(remainingNames, r.GetName())
			}
			req.ElementsMatch(tc.expectedRemainingNames, remainingNames)
		})
	}
}

func TestRemoveTransformerPlugin_Config_WithSelector(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()

	// Verify Config with selector fills the Targets field correctly
	config := []byte(`
targets:
  - kind: ConfigMap
    name: test
    namespace: default
`)
	err := plugin.Config(nil, config)
	req.NoError(err)

	// Verify we can cast and inspect
	p, ok := plugin.(*extras.RemoveTransformerPlugin)
	req.True(ok)
	req.Len(p.Targets, 1)
	req.Equal("ConfigMap", p.Targets[0].Kind)
	req.Equal("test", p.Targets[0].Name)

	_ = types.Selector{}
}
