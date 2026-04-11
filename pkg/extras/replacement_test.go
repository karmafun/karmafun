package extras_test

// cSpell: words myrepo failf filesys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

const (
	TargetConfigMapName  = "target-cm"
	TargetConfigMapName2 = "target-cm2"

	sourceCMValueYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: replaced-value
`

	targetCMOriginalTargetValueYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original-value
`

	configBasicReplacement = `
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
`
	configBasicReplacementWithPath = `
replacements:
  - path: simple.yaml
`

	configNilSourceFieldPath = `
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
`
)

type transformTestCase struct {
	assert    func(*require.Assertions, resmap.ResMap)
	files     map[string]string
	name      string
	config    string
	wantErr   string
	resources []string
}

func mustConfiguredReplacementPlugin(t *testing.T, config string, fSys filesys.FileSystem) resmap.TransformerPlugin {
	t.Helper()
	req := require.New(t)
	if fSys == nil {
		fSys = filesys.MakeFsOnDisk()
	}

	helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	err = plugin.Config(helpers, []byte(config))
	req.NoError(err)

	return plugin
}

func fieldValueByName(req *require.Assertions, rm resmap.ResMap, name, fieldPath string) any {
	for _, r := range rm.Resources() {
		if r.GetName() == name {
			value, err := r.GetFieldValue(fieldPath)
			req.NoError(err)
			return value
		}
	}

	req.Failf("resource not found", "resource %q not found", name)
	return nil
}

func runTransformCase(t *testing.T, tc *transformTestCase) {
	t.Helper()
	req := require.New(t)
	fSys := filesys.MakeFsOnDisk()
	if tc.files != nil {
		fSys = filesys.MakeFsInMemory()

		for name, content := range tc.files {
			err := fSys.WriteFile(name, []byte(content))
			req.NoError(err)
		}
	}
	plugin := mustConfiguredReplacementPlugin(t, tc.config, fSys)
	rm := makeResMap(t, tc.resources...)

	err := plugin.Transform(rm)
	if tc.wantErr != "" {
		req.Error(err)
		req.Contains(err.Error(), tc.wantErr)
		return
	}

	req.NoError(err)
	if tc.assert != nil {
		tc.assert(req, rm)
	}
}

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
	err = plugin.Config(helpers, []byte(configBasicReplacement))
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

func TestExtendedReplacementTransformerPlugin_Config_ConflictPathAndInline(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	err = plugin.Config(helpers, []byte(`
replacements:
  - path: some/path
    source:
      kind: ConfigMap
      name: test
    targets:
      - select:
          kind: ConfigMap
`))
	req.Error(err)
	req.Contains(err.Error(), "cannot specify both path and inline replacement")
}

func TestExtendedReplacementTransformerPlugin_Config_WithSourceNoField(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewExtendedReplacementTransformerPlugin()
	err = plugin.Config(helpers, []byte(configNilSourceFieldPath))
	req.NoError(err)
}

func TestExtendedReplacementTransformerPlugin_Transform_EmptySource(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := mustConfiguredReplacementPlugin(t, `
replacements: []
`, nil)
	rm := makeResMap(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`)

	err := plugin.Transform(rm)
	req.NoError(err)
}

func TestExtendedReplacementTransformerPlugin_Transform_SuccessCases(t *testing.T) {
	t.Parallel()

	testCases := []transformTestCase{
		{
			name:      "basic replacement",
			config:    configBasicReplacement,
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("replaced-value", fieldValueByName(req, rm, TargetConfigMapName, "data.targetValue"))
			},
		},
		{
			name:   "basic replacement with path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": `
source:
    kind: ConfigMap
    name: source-cm
    fieldPath: data.value
targets:
    - select:
        kind: ConfigMap
        name: target-cm
      fieldPaths:
        - data.targetValue
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("replaced-value", fieldValueByName(req, rm, TargetConfigMapName, "data.targetValue"))
			},
		},
		{
			name:   "basic replacement with path and slice",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": `
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
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("replaced-value", fieldValueByName(req, rm, TargetConfigMapName, "data.targetValue"))
			},
		},
		{
			name: "source delimiter",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  version: "myrepo/v1.2.3"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  tag: original
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("v1.2.3", fieldValueByName(req, rm, TargetConfigMapName, "data.tag"))
			},
		},
		{
			name: "source encoding",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: "hello"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  hexValue: original
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("68656c6c6f", fieldValueByName(req, rm, TargetConfigMapName, "data.hexValue"))
			},
		},
		{
			name: "label selector",
			config: `
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
`,
			resources: []string{sourceCMValueYAML, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
  labels:
    env: prod
data:
  targetValue: original-value
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm2
  labels:
    env: dev
data:
  targetValue: original-value
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("replaced-value", fieldValueByName(req, rm, TargetConfigMapName, "data.targetValue"))
				req.Equal("original-value", fieldValueByName(req, rm, TargetConfigMapName2, "data.targetValue"))
			},
		},
		{
			name: "create option",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: created-value
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  existingField: exists
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("created-value", fieldValueByName(req, rm, TargetConfigMapName, "data.newField"))
			},
		},
		{
			name: "reject selector",
			config: `
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
`,
			resources: []string{sourceCMValueYAML, `apiVersion: v1
kind: ConfigMap
metadata:
  name: included-cm
data:
  targetValue: original
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: excluded-cm
data:
  targetValue: original
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("replaced-value", fieldValueByName(req, rm, "included-cm", "data.targetValue"))
				req.Equal("original", fieldValueByName(req, rm, "excluded-cm", "data.targetValue"))
			},
		},
		{
			name: "target delimiter prefix",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  prefix: "prefix"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: "original/path"
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("prefix/original/path", fieldValueByName(req, rm, TargetConfigMapName, "data.value"))
			},
		},
		{
			name: "target delimiter suffix",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  suffix: "appended"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: "original/path"
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("original/path/appended", fieldValueByName(req, rm, TargetConfigMapName, "data.value"))
			},
		},
		{
			name: "non scalar target",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key1: value1
  key2: value2
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  oldKey: oldValue
`},
			assert: func(req *require.Assertions, rm resmap.ResMap) {
				req.Equal("value1", fieldValueByName(req, rm, TargetConfigMapName, "data.key1"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runTransformCase(t, &tc)
		})
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_ErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []transformTestCase{
		{
			name: "source missing",
			config: `
replacements:
  - targets:
      - select:
          kind: ConfigMap
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
`},
			wantErr: "must specify a source",
		},
		{
			name: "create with wildcard target",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: test-value
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  existingKey: exists
`},
			wantErr: "multi-value target",
		},
		{
			name: "no target selector",
			config: `
replacements:
  - source:
      kind: ConfigMap
      name: source-cm
      fieldPath: data.value
    targets:
      - fieldPaths:
          - data.targetValue
`,
			resources: []string{sourceCMValueYAML},
			wantErr:   "target must specify resources to select",
		},
		{
			name: "delimiter on non scalar source",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key: value
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`},
			wantErr: "scalar",
		},
		{
			name: "source fieldPath missing",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: existing
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: original
`},
			wantErr: "is missing for replacement source",
		},
		{
			name: "source not found",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  value: original
`},
			wantErr: "nothing selected",
		},
		{
			name: "delimiter with extension path",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  value: "hello/world"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  embedded: |
    nested:
      key: original
`},
			wantErr: "delimiter option cannot be used with extensions",
		},
		{
			name: "source delimiter out of bounds",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  version: "only-one-part"
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  tag: original
`},
			wantErr: "out of bounds",
		},
		{
			name: "encoding on non scalar source",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
data:
  key: value
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`},
			wantErr: "scalar",
		},
		{
			name: "multiple source matches",
			config: `
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
`,
			resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm1
data:
  value: value1
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm2
data:
  value: value2
`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: target-cm
data:
  targetValue: original
`},
			wantErr: "multiple matches",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runTransformCase(t, &tc)
		})
	}
}

func TestExtendedReplacementTransformerPlugin_Transform_PreviousIds(t *testing.T) {
	t.Parallel()

	tc := transformTestCase{
		name:   "previous IDs",
		config: configBasicReplacement,
		resources: []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: source-cm
  annotations:
    config.kubernetes.io/previousNames: old-source-cm
    config.kubernetes.io/previousNamespaces: default
    config.kubernetes.io/previousKinds: ConfigMap
data:
  value: replaced-value
`, targetCMOriginalTargetValueYAML},
	}

	runTransformCase(t, &tc)
}

func TestExtendedReplacementTransformerPlugin_Types(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	selector := types.Selector{}
	req.Empty(selector.Name)
}

func TestExtendedReplacementTransformerPlugin_BadPathConfiguration(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	testCases := []transformTestCase{
		{
			name:   "bad path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple2.yaml": `
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
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			wantErr:   "while loading replacement path",
		},
		{
			name:   "bad yaml in path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": `
- source:
kind: ConfigMap
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			wantErr:   "while unmarshaling replacement path",
		},
		{
			name:   "bad slice replacement path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": `
- source:
    kind: ConfigMap
    name:
     - one
     - two
    fieldPath: data.value
  targets:
    - select:
        kind: ConfigMap
        name: target-cm
      fieldPaths:
        - data.targetValue
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			wantErr:   "while unmarshaling slice replacement path",
		},
		{
			name:   "bad simple replacement path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": `
source:
  kind: ConfigMap
  name:
    - one
    - two
  fieldPath: data.value
targets:
- select:
    kind: ConfigMap
    name: target-cm
  fieldPaths:
    - data.targetValue
`,
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			wantErr:   "while unmarshaling simple replacement path",
		},
		{
			name:   "bad yaml replacement path",
			config: configBasicReplacementWithPath,
			files: map[string]string{
				"simple.yaml": "",
			},
			resources: []string{sourceCMValueYAML, targetCMOriginalTargetValueYAML},
			wantErr:   "unsupported replacement type encountered",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fSys := filesys.MakeFsInMemory()

			if tc.files != nil {
				for name, content := range tc.files {
					err := fSys.WriteFile(name, []byte(content))
					req.NoError(err)
				}
			}
			helpers, err := plugins.NewPluginHelpersInFileSystem(fSys)
			req.NoError(err)

			plugin := extras.NewExtendedReplacementTransformerPlugin()
			err = plugin.Config(helpers, []byte(tc.config))
			req.Error(err)
			if tc.wantErr != "" {
				req.Contains(err.Error(), tc.wantErr)
			}
		})
	}
}
