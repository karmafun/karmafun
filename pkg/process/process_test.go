// cSpell: words processpkg fcfg
package process_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	processpkg "github.com/karmafun/karmafun/pkg/process"
	"github.com/karmafun/karmafun/pkg/utils"
)

type mockBuilder struct {
	buildFn func(resid.Gvk) (resmap.Configurable, error)
}

func (m mockBuilder) Build(id resid.Gvk) (resmap.Configurable, error) {
	return m.buildFn(id)
}

var _ processpkg.PluginBuilder = (*mockBuilder)(nil)

type testTransformerPlugin struct {
	configErr    error
	transformErr error
	lastConfig   string
	configCalls  int
}

func (p *testTransformerPlugin) Config(_ *resmap.PluginHelpers, c []byte) error {
	p.configCalls++
	p.lastConfig = string(c)
	return p.configErr
}

func (p *testTransformerPlugin) Transform(m resmap.ResMap) error {
	if p.transformErr != nil {
		return p.transformErr
	}
	for _, r := range m.Resources() {
		a := r.GetAnnotations()
		if a == nil {
			a = map[string]string{}
		}
		a["transformed"] = "true"
		if err := r.SetAnnotations(a); err != nil {
			return fmt.Errorf("setting annotations: %w", err)
		}
	}
	return nil
}

type testGeneratorPlugin struct {
	configErr   error
	generateErr error
	configCalls int
}

func (p *testGeneratorPlugin) Config(_ *resmap.PluginHelpers, _ []byte) error {
	p.configCalls++
	return p.configErr
}

func (p *testGeneratorPlugin) Generate() (resmap.ResMap, error) {
	if p.generateErr != nil {
		return nil, p.generateErr
	}
	rm := resmap.New()
	node := mustNode(`apiVersion: v1
kind: ConfigMap
metadata:
  name: generated
`)
	if err := rm.Append(&resource.Resource{RNode: *node}); err != nil {
		return nil, fmt.Errorf("appending generated resource: %w", err)
	}
	return rm, nil
}

type functionConfigurableTransformer struct {
	configureErr    error
	capturedKind    string
	capturedVersion string
	configureCalls  int
	configCalls     int
}

func (p *functionConfigurableTransformer) ConfigureWithFunctionConfig(
	_ *resmap.PluginHelpers,
	functionConfig *yaml.RNode,
) error {
	p.configureCalls++
	p.capturedKind = functionConfig.GetKind()
	p.capturedVersion = functionConfig.GetApiVersion()
	return p.configureErr
}

func (p *functionConfigurableTransformer) Config(_ *resmap.PluginHelpers, _ []byte) error {
	p.configCalls++
	return nil
}

func (p *functionConfigurableTransformer) Transform(_ resmap.ResMap) error {
	return nil
}

func mustNode(content string) *yaml.RNode {
	n, err := yaml.Parse(content)
	if err != nil {
		panic(err)
	}
	return n
}

func TestProcess_TransformerAndGeneratorFlows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		builder    processpkg.PluginBuilder
		assertions func(*testing.T, *framework.ResourceList)
		name       string
		cfg        string
		wantErr    string
		items      []string
	}{
		{
			name: "transformer success with cleanup and prune local",
			cfg: `apiVersion: builtin
kind: AnyTransformer
metadata:
  name: cfg
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.karmafun.dev/prune-local: "true"
`,
			items: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: keep
  annotations:
    config.kustomize.io/id: keep
data:
  key: value
`,
				`apiVersion: v1
kind: Secret
metadata:
  name: prune
  annotations:
    config.karmafun.dev/local-config: "true"
`,
			},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &testTransformerPlugin{}, nil
			}},
			assertions: func(t *testing.T, rl *framework.ResourceList) {
				t.Helper()
				req := require.New(t)
				req.Len(rl.Items, 1)
				req.Equal("keep", rl.Items[0].GetName())
				a := rl.Items[0].GetAnnotations()
				_, hasBuild := a[utils.BuildAnnotationsRefBy]
				req.False(hasBuild)
			},
		},
		{
			name: "generator success appends and transfers annotations",
			cfg: `apiVersion: builtin
kind: AnyGenerator
metadata:
  name: cfg
  annotations:
    config.karmafun.dev/path: generated.yaml
`,
			items: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: existing
`,
			},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &testGeneratorPlugin{}, nil
			}},
			assertions: func(t *testing.T, rl *framework.ResourceList) {
				t.Helper()
				req := require.New(t)
				req.Len(rl.Items, 2)
				req.Equal("generated", rl.Items[1].GetName())
				a := rl.Items[1].GetAnnotations()
				req.Equal("generated.yaml", a["internal.config.kubernetes.io/path"])
			},
		},
		{
			name: "inject local when plugin is missing",
			cfg: `apiVersion: builtin
kind: MissingPlugin
metadata:
  name: cfg
  annotations:
    config.karmafun.dev/inject-local: "true"
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return nil, errors.New("plugin unavailable")
			}},
			assertions: func(t *testing.T, rl *framework.ResourceList) {
				t.Helper()
				req := require.New(t)
				req.Len(rl.Items, 1)
				req.Equal("cfg", rl.Items[0].GetName())
			},
		},
		{
			name: "missing plugin without inject local returns error",
			cfg: `apiVersion: builtin
kind: MissingPlugin
metadata:
  name: cfg
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return nil, errors.New("not found")
			}},
			wantErr: "creating plugin: not found",
		},
		{
			name: "config error from plugin",
			cfg: `apiVersion: builtin
kind: AnyTransformer
metadata:
  name: cfg
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &testTransformerPlugin{configErr: errors.New("bad config")}, nil
			}},
			wantErr: "fails configuration: bad config",
		},
		{
			name: "transform error",
			cfg: `apiVersion: builtin
kind: AnyTransformer
metadata:
  name: cfg
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &testTransformerPlugin{transformErr: errors.New("boom")}, nil
			}},
			wantErr: "while transforming resources: transforming resources: boom",
		},
		{
			name: "generate error",
			cfg: `apiVersion: builtin
kind: AnyGenerator
metadata:
  name: cfg
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &testGeneratorPlugin{generateErr: errors.New("gen fail")}, nil
			}},
			wantErr: "while generating resources: generating resource(s): gen fail",
		},
		{
			name: "plugin neither generator nor transformer",
			cfg: `apiVersion: builtin
kind: Any
metadata:
  name: cfg
`,
			items: []string{},
			builder: mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
				return &configOnlyPlugin{}, nil
			}},
			wantErr: "is neither a generator nor a transformer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			rl := &framework.ResourceList{FunctionConfig: mustNode(tt.cfg)}
			for _, item := range tt.items {
				rl.Items = append(rl.Items, mustNode(item))
			}

			p := processpkg.NewProcessor(tt.builder)
			err := p.Process(rl)

			if tt.wantErr != "" {
				req.Error(err)
				req.Contains(err.Error(), tt.wantErr)
				return
			}

			req.NoError(err)
			if tt.assertions != nil {
				tt.assertions(t, rl)
			}
		})
	}
}

type configOnlyPlugin struct{}

func (p *configOnlyPlugin) Config(_ *resmap.PluginHelpers, _ []byte) error {
	return nil
}

func TestProcess_UsesFunctionConfigConfigurableWhenAvailable(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	pl := &functionConfigurableTransformer{}
	p := processpkg.NewProcessor(mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
		return pl, nil
	}})

	rl := &framework.ResourceList{
		FunctionConfig: mustNode(`apiVersion: builtin
kind: MyTransformer
metadata:
  name: cfg
`),
	}

	err := p.Process(rl)
	req.NoError(err)
	req.Equal(1, pl.configureCalls)
	req.Equal(0, pl.configCalls)
	req.Equal("MyTransformer", pl.capturedKind)
	req.Equal("builtin", pl.capturedVersion)
}

func TestProcess_FunctionConfigConfigurableError(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	pl := &functionConfigurableTransformer{configureErr: errors.New("fcfg fail")}
	p := processpkg.NewProcessor(mockBuilder{buildFn: func(_ resid.Gvk) (resmap.Configurable, error) {
		return pl, nil
	}})

	rl := &framework.ResourceList{
		FunctionConfig: mustNode(`apiVersion: builtin
kind: MyTransformer
metadata:
  name: cfg
`),
	}

	err := p.Process(rl)
	req.Error(err)
	req.Contains(err.Error(), "configuring plugin with function config: fcfg fail")
	req.Equal(1, pl.configureCalls)
	req.Equal(0, pl.configCalls)
}

func TestNewDefaultProcessor(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	p := processpkg.NewDefaultProcessor()
	req.NotNil(p)
}
