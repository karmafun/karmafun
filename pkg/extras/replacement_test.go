package extras_test

// cSpell: words karmafun

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestNewExtendedReplacementTransformerPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	req.NotNil(plugin)
}

func TestExtendedReplacementTransformerPlugin_Config(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestExtendedReplacementTransformerPlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	err = plugin.Config(helpers, []byte("{invalid yaml"))
	req.Error(err)
}

func TestExtendedReplacementTransformerPlugin_Transform_BasicReplacement(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: replaced-value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original-value
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	resources := rm.Resources()
	var targetResource *resource.Resource
	for _, r := range resources {
		if r.GetName() == "target-cm" {
			targetResource = r
			break
		}
	}
	req.NotNil(targetResource)

	value, err := targetResource.GetFieldValue("data.targetValue")
	req.NoError(err)
	req.Equal("replaced-value", value)
}

func TestExtendedReplacementTransformerPlugin_Transform_EmptySource(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements: []
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	rm := makeResMap(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`)

	err = plugin.Transform(rm)
	req.NoError(err)
}

func TestExtendedReplacementTransformerPlugin_Transform_WithDelimiter(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.version
      options:
        delimiter: "/"
        index: 1
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.tag
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  version: "myrepo/v1.2.3"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  tag: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	resources := rm.Resources()
	var targetResource *resource.Resource
	for _, r := range resources {
		if r.GetName() == "target-cm" {
			targetResource = r
			break
		}
	}
	req.NotNil(targetResource)

	value, err := targetResource.GetFieldValue("data.tag")
	req.NoError(err)
	req.Equal("v1.2.3", value)
}

func TestExtendedReplacementTransformerPlugin_Transform_WithEncoding(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
      options:
        encoding: "hex"
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.hexValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: "hello"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  hexValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	resources := rm.Resources()
	var targetResource *resource.Resource
	for _, r := range resources {
		if r.GetName() == "target-cm" {
			targetResource = r
			break
		}
	}
	req.NotNil(targetResource)

	value, err := targetResource.GetFieldValue("data.hexValue")
	req.NoError(err)
	req.Equal("68656c6c6f", value)
}

func TestExtendedReplacementTransformerPlugin_Config_ConflictPathAndInline(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - path: some/path
    source:
      kind: ConfigMap
      name: test
    targets:
      - select:
          kind: ConfigMap
`)
	err = plugin.Config(helpers, config)
	req.Error(err)
	req.Contains(err.Error(), "cannot specify both path and inline replacement")
}

func TestExtendedFilter_NoSourceAndNoTargets(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Test with a nil source in the replacement list to trigger the filter error
	config := []byte(`
replacements:
  - targets:
      - select:
          kind: ConfigMap
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	rm := makeResMap(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
`)
	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "must specify a source")
}

func TestExtendedReplacementTransformerPlugin_Transform_WithLabelSelector(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          labelSelector: "env=prod"
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: replaced-value
`)
	req.NoError(err)

	// Target with matching label
	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
  labels:
    env: prod
data:
  targetValue: original-value
`)
	req.NoError(err)

	// Target without matching label (should not be replaced)
	targetNode2, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm2
  labels:
    env: dev
data:
  targetValue: original-value
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode2}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "source-cm" {
			continue
		}
		v, err2 := r.GetFieldValue("data.targetValue")
		req.NoError(err2)
		if r.GetName() == "target-cm" {
			req.Equal("replaced-value", v)
		} else if r.GetName() == "target-cm2" {
			req.Equal("original-value", v)
		}
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_WithCreate(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.newField
        options:
          create: true
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: created-value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  existingField: exists
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "target-cm" {
			v, err2 := r.GetFieldValue("data.newField")
			req.NoError(err2)
			req.Equal("created-value", v)
		}
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_WithReject(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
        reject:
          - name: excluded-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: replaced-value
`)
	req.NoError(err)

	includedNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: included-cm
data:
  targetValue: original
`)
	req.NoError(err)

	excludedNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: excluded-cm
data:
  targetValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *includedNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *excludedNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "source-cm" {
			continue
		}
		v, err2 := r.GetFieldValue("data.targetValue")
		req.NoError(err2)
		if r.GetName() == "included-cm" {
			req.Equal("replaced-value", v)
		} else if r.GetName() == "excluded-cm" {
			req.Equal("original", v)
		}
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_PreviousIds(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	// Source node with previousNames annotation
	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
  annotations:
    config.kubernetes.io/previousNames: old-source-cm
    config.kubernetes.io/previousNamespaces: default
    config.kubernetes.io/previousKinds: ConfigMap
data:
  value: replaced-value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original-value
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)
}

func TestShouldCreateField_WithWildcard(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Use wildcard path with create option to trigger error
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.*
        options:
          create: true
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: test-value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  existingKey: exists
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "multi-value target")
}

func TestExtendedReplacementTransformerPlugin_Config_WithSourceNoField(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Source with no fieldPath uses the default
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestExtendedReplacementTransformerPlugin_Transform_NoTargetSelector(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Target with no Select
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: replaced-value
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "target must specify resources to select")
}

func TestExtendedReplacementTransformerPlugin_Transform_DelimiterOnNonScalar(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Delimiter option on source field that is not scalar
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data
      options:
        delimiter: "/"
        index: 0
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key: value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	// Delimiter on non-scalar source should error
	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "scalar")
}

func TestExtendedReplacementTransformerPlugin_SourceFieldPath_Missing(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.nonexistent
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.value
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: existing
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "is missing for replacement source")
}

func TestExtendedReplacementTransformerPlugin_MultipleSourcesNotFound(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: nonexistent-source
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.value
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "nothing selected")
}

func TestSetFieldValue_DelimiterWithExtension(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	// Using delimiter with an extended path should error
	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.embedded.!!yaml.nested.key
        options:
          delimiter: "/"
          index: 0
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: "hello/world"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  embedded: |
    nested:
      key: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "delimiter option cannot be used with extensions")
}

func TestExtendedRefinedValue_DelimiterOutOfBounds(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.version
      options:
        delimiter: "/"
        index: 10
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.tag
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  version: "only-one-part"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  tag: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "out of bounds")
}

func TestRefinedValue_EncodingOnNonScalar(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	// Encoding on non-scalar should fail
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data
      options:
        encoding: "base64"
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key: value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "scalar")
}

func TestExtendedReplacementTransformerPlugin_Transform_MultipleSourcesError(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	// Multiple source candidates should cause an error
	sourceNode1, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm1
data:
  value: value1
`)
	req.NoError(err)

	sourceNode2, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm2
data:
  value: value2
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode1}))
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode2}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.Error(err)
	req.Contains(err.Error(), "multiple matches")
}

// Test using the _ field to add field option for delimiter with prefix
func TestExtendedReplacementTransformerPlugin_Transform_DelimiterPrefix(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.prefix
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.value
        options:
          delimiter: "/"
          index: -1
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  prefix: "prefix"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: "original/path"
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "target-cm" {
			v, err2 := r.GetFieldValue("data.value")
			req.NoError(err2)
			req.Equal("prefix/original/path", v)
		}
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_DelimiterSuffix(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.suffix
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.value
        options:
          delimiter: "/"
          index: 100
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  suffix: "appended"
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: "original/path"
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "target-cm" {
			v, err2 := r.GetFieldValue("data.value")
			req.NoError(err2)
			req.Equal("original/path/appended", v)
		}
	}
}

// Test getRefinedValue with nil options returns the value directly
func TestRefinedValue_NilOptions(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data.targetValue
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: simple-value
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "target-cm" {
			v, err2 := r.GetFieldValue("data.targetValue")
			req.NoError(err2)
			req.Equal("simple-value", v)
		}
	}
}

// Test setFieldValue when target is a non-scalar (mapping) node but no extensions
func TestExtendedReplacementTransformerPlugin_Transform_NonScalarTarget(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	config := []byte(`
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data
    targets:
      - select:
          kind: ConfigMap
          name: target-cm
        fieldPaths:
          - data
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	sourceNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key1: value1
  key2: value2
`)
	req.NoError(err)

	targetNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  oldKey: oldValue
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *sourceNode}))
	req.NoError(rm.Append(&resource.Resource{RNode: *targetNode}))

	err = plugin.Transform(rm)
	req.NoError(err)

	for _, r := range rm.Resources() {
		if r.GetName() == "target-cm" {
			v, err2 := r.GetFieldValue("data.key1")
			req.NoError(err2)
			req.Equal("value1", v)
		}
	}
}

// test using types
func TestExtendedReplacementTransformerPlugin_Types(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Verify the types used in tests are accessible
	selector := types.Selector{}
	req.Empty(selector.Name)
}
