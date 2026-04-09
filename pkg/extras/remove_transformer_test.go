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

func TestRemoveTransformerPlugin_Transform_RemovesMatching(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	config := []byte(`
targets:
  - kind: ConfigMap
    name: to-remove
`)
	err := plugin.Config(nil, config)
	req.NoError(err)

	rm := makeResMap(t,
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
	)
	req.Equal(2, rm.Size())

	err = plugin.Transform(rm)
	req.NoError(err)
	req.Equal(1, rm.Size())
	// The remaining resource should be the one we didn't remove
	resources := rm.Resources()
	req.Equal("keep-me", resources[0].GetName())
}

func TestRemoveTransformerPlugin_Transform_MultipleTargets(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	config := []byte(`
targets:
  - kind: ConfigMap
    name: remove1
  - kind: ConfigMap
    name: remove2
`)
	err := plugin.Config(nil, config)
	req.NoError(err)

	rm := makeResMap(t,
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
	)
	req.Equal(3, rm.Size())

	err = plugin.Transform(rm)
	req.NoError(err)
	req.Equal(1, rm.Size())
}

func TestRemoveTransformerPlugin_Transform_NoMatchingResources(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	config := []byte(`
targets:
  - kind: ConfigMap
    name: nonexistent
`)
	err := plugin.Config(nil, config)
	req.NoError(err)

	rm := makeResMap(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: keep-me
`)
	req.Equal(1, rm.Size())

	err = plugin.Transform(rm)
	req.NoError(err)
	// Nothing should be removed
	req.Equal(1, rm.Size())
}

func TestRemoveTransformerPlugin_Transform_ByKindOnly(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewRemoveTransformerPlugin()
	config := []byte(`
targets:
  - kind: Secret
`)
	err := plugin.Config(nil, config)
	req.NoError(err)

	rm := makeResMap(t,
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
	)
	req.Equal(2, rm.Size())

	err = plugin.Transform(rm)
	req.NoError(err)
	req.Equal(1, rm.Size())

	resources := rm.Resources()
	req.Equal("keep-cm", resources[0].GetName())
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
