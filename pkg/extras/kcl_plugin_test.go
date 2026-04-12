package extras_test

// cSpell: words karmafun kcl paralleltest filesys

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
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
apiVersion: karmafun.dev/v1alpha1
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
	req.Contains(err.Error(), "while unmarshaling KCLRunnerConfig")
}

func TestKCLBasePlugin_ConfigureWithFunctionConfig(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	config := `
apiVersion: karmafun.dev/v1alpha1
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
apiVersion: karmafun.dev/v1alpha1
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

func TestKCLGeneratorPlugin_WithParamResources_NonExistent(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	fSys := filesys.MakeFsInMemory()
	helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
	req.NoError(err)

	plugin := &extras.KCLGeneratorPlugin{}
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
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

	_, err = plugin.Generate()
	req.Error(err)
	req.Contains(err.Error(), "while loading param resource from path param-resource.yaml")
}

func TestKCLGeneratorPlugin_WithParamResources_ErrorLoadingResource(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	fSys := filesys.MakeFsInMemory()

	// This will generate an error because there is no existing resource to replace. Tricky.
	paramResourceContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: param-resource
  annotations:
    internal.config.kubernetes.io/generatorBehavior: replace
data:
  inputKey: inputValue

`

	err := fSys.WriteFile("param-resource.yaml", []byte(paramResourceContent))
	req.NoError(err)

	helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
	req.NoError(err)

	plugin := &extras.KCLGeneratorPlugin{}
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
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

	_, err = plugin.Generate()
	req.Error(err)
	req.Contains(err.Error(), "while absorbing param resources from path")
}

func TestKCLGeneratorPlugin_WithParamResources_LoadsParamResource(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	fSys := filesys.MakeFsInMemory()

	paramResourceContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: param-resource
data:
  inputKey: inputValue

`

	err := fSys.WriteFile("param-resource.yaml", []byte(paramResourceContent))
	req.NoError(err)

	helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
	req.NoError(err)

	plugin := &extras.KCLGeneratorPlugin{}
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
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

const configmapYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
  namespace: default
data:
  key: value
`

func TestKCLTransformerPlugin_Transform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		files     map[string]string
		assert    func(*require.Assertions, resmap.ResMap)
		name      string
		config    string
		wantErr   string
		want      string
		resources []string
	}{
		{
			name: "nop transformation",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    # Pass items through without modification
    _items = option("items")
`,
			resources: []string{configmapYaml},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				// Assert that the input resource is unchanged
				r, err := rm.GetById(
					resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "ConfigMap"), "example-configmap", "default"),
				)
				req.NoError(err)
				keyValue, err := r.Pipe(yaml.Lookup("data", "key"))
				req.NoError(err)
				req.NotNil(keyValue)
				req.Equal("value", keyValue.YNode().Value)
			},
		},
		{
			name: "change key value",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    _item = option("items")[0]
    _item.data.key = "newValue"

    _item.metadata.annotations = {
        'internal.config.kubernetes.io/generatorBehavior' = "replace"
    }
    [_item]
`,
			resources: []string{configmapYaml},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				// Assert that the input resource is unchanged
				r, err := rm.GetById(
					resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "ConfigMap"), "example-configmap", "default"),
				)
				req.NoError(err)
				keyValue, err := r.Pipe(yaml.Lookup("data", "key"))
				req.NoError(err)
				req.NotNil(keyValue)
				req.Equal("newValue", keyValue.YNode().Value)
			},
		},
		{
			name: "Error on loading resources",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
param_resources:
  - param-resource.yaml
spec:
  source: |
    # Pass items through without modification
    _items = option("items")
`,
			resources: []string{configmapYaml},
			wantErr:   "while preparing function configuration",
		},
		{
			name: "Error on transformation",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    _item = option("items")[1]
`,
			resources: []string{configmapYaml},
			wantErr:   "while transforming resources:",
		},
		{
			name: "Bad return values",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    [{badReturn: "true"}]
`,
			resources: []string{configmapYaml},
			wantErr:   "while creating resmap from nodes",
		},
		{
			name: "Error on absorbing result resources",
			config: `
apiVersion: karmafun.dev/v1alpha1
kind: KCLRun
metadata:
  name: test-kcl
spec:
  source: |
    _item = option("items")[0]
    _item.data.key = "newValue"

    [_item]
`,
			resources: []string{configmapYaml},
			wantErr:   "while absorbing transformed resources",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			fSys := filesys.MakeFsInMemory()
			for path, content := range tt.files {
				err := fSys.WriteFile(path, []byte(content))
				req.NoError(err, "writing file %s should not error", path)
			}
			helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
			req.NoError(err)

			plugin := extras.NewKCLTransformerPlugin()
			err = plugin.Config(helpers, []byte(tt.config))
			req.NoError(err)

			var rm resmap.ResMap
			if len(tt.resources) == 0 {
				// Default to a simple resource if none provided, since the plugin expects some input.
				rm = makeResMap(t)
			} else {
				rm = makeResMap(t, tt.resources...)
			}

			err = plugin.Transform(rm)
			if tt.wantErr != "" {
				req.Error(err)
				req.Contains(err.Error(), tt.wantErr)
			} else {
				req.NoError(err)
				if tt.assert != nil {
					tt.assert(req, rm)
				}
				if tt.want != "" {
					var b bytes.Buffer
					err = kio.ByteWriter{Writer: &b}.Write(rm.ToRNodeSlice())
					req.NoError(err, "writing generated resources should not error")
					req.Equal(tt.want, b.String(), "generated resources should match expected output")
				}
			}
		})
	}
}

func TestKCLGeneratorPlugin_Generate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string // description of this test case
		functionConfig string // the functionConfig for the plugin
		want           string // the expected output yaml string
		wantErr        bool
	}{
		{
			name: "generate a configmap with KCL code",
			functionConfig: `
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example-kcl-run
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "example-configmap"
      metadata.namespace = "default"
      data.key = "value"
    }
`,
			want:    configmapYaml,
			wantErr: false,
		},
		{
			name: "Configmap with source file",
			functionConfig: `
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example-kcl-run
spec:
  source: ../../tests/kcl/configmap.k
`,
			want:    configmapYaml,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			helpers, err := plugins.NewPluginHelpers()
			req.NoError(err, "creating plugin helpers should not error")

			// TODO: construct the receiver type.
			var p extras.KCLGeneratorPlugin
			err = p.Config(helpers, []byte(tt.functionConfig))
			req.NoError(err, "configuring plugin should not error")
			got, gotErr := p.Generate()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Generate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Generate() succeeded unexpectedly")
			}
			var b bytes.Buffer
			req.NoError(err, "encoding generated resources should not error")
			err = kio.ByteWriter{Writer: &b}.Write(got.ToRNodeSlice())
			req.NoError(err, "writing generated resources should not error")
			req.Equal(tt.want, b.String(), "generated resources should match expected output")
		})
	}
}
