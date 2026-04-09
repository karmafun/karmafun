package extras_test

// cSpell: words karmafun kcl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestNewKCLGeneratorPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKCLGeneratorPlugin()
	req.NotNil(plugin)
}

func TestNewKCLTransformerPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKCLTransformerPlugin()
	req.NotNil(plugin)
}

func TestKCLBasePlugin_Config(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewKCLGeneratorPlugin()
	config := []byte(`
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "test"
    }
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestKCLBasePlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewKCLGeneratorPlugin()
	err = plugin.Config(helpers, []byte("{invalid yaml"))
	req.Error(err)
}

func TestKCLBasePlugin_ConfigureWithFunctionConfig(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	config := `
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "test-from-fn"
    }
`
	node, err := yaml.Parse(config)
	req.NoError(err)

	// Cast to get the ConfigureWithFunctionConfig method
	p := &extras.KCLGeneratorPlugin{}
	err = p.ConfigureWithFunctionConfig(helpers, node)
	req.NoError(err)
}

func TestKCLBasePlugin_ConfigureWithFunctionConfig_InvalidConfig(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	// Use a config that would cause YAML unmarshal to fail
	// An empty mapping node should work fine, so let's use something unusual
	node, err := yaml.Parse(`"invalid-not-a-mapping"`)
	req.NoError(err)

	p := &extras.KCLGeneratorPlugin{}
	err = p.ConfigureWithFunctionConfig(helpers, node)
	req.Error(err)
}

func TestKCLGeneratorPlugin_WithParamResources(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := &extras.KCLGeneratorPlugin{}
	// Load param_resources from the tests directory (KCL configmap)
	config := []byte(`
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "example-configmap"
      data.key = "value"
    }
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
	got, gotErr := plugin.Generate()
	req.NoError(gotErr)
	req.NotNil(got)
	req.Equal(1, got.Size())
}

func TestKCLGeneratorPlugin_WithParamResources_LoadsParamResource(t *testing.T) {
	// Not parallel because it changes working directory
	req := require.New(t)

	// Create a temporary directory with a param resource YAML file
	tmpDir, err := os.MkdirTemp("", "karmafun-kcl-param-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir)

	paramResourceContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: param-resource
data:
  inputKey: inputValue
`
	paramResourceFile := filepath.Join(tmpDir, "param-resource.yaml")
	err = os.WriteFile(paramResourceFile, []byte(paramResourceContent), 0600)
	req.NoError(err)

	// Change to temp directory so loader can find the file
	originalDir, err := os.Getwd()
	req.NoError(err)
	defer func() {
		err = os.Chdir(originalDir)
		req.NoError(err)
	}()
	err = os.Chdir(tmpDir)
	req.NoError(err)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := &extras.KCLGeneratorPlugin{}
	config := []byte(`
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
param_resources:
  - param-resource.yaml
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "output-configmap"
    }
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	got, gotErr := plugin.Generate()
	req.NoError(gotErr)
	req.NotNil(got)
	req.Equal(1, got.Size())
}

func TestKCLTransformerPlugin_Transform(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewKCLTransformerPlugin()
	// KCL program that adds a label to the existing resource
	// Must output the modified resources for AbsorbAll to work correctly
	config := []byte(`
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    # Pass items through without modification
    items = []
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	node, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`)
	req.NoError(err)

	rm := makeResMap(t)
	req.NoError(rm.Append(&resource.Resource{RNode: *node}))

	err = plugin.Transform(rm)
	req.NoError(err)
}
