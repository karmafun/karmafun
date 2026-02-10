package extras

// cSpell: words kcl

import (
	"fmt"

	"kcl-lang.io/krm-kcl/pkg/config"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KCLPluginRun struct {
	config.KCLRun  `         json:",inline"         yaml:",inline"`
	ParamResources []string `json:"param_resources" yaml:"param_resources"`
}

type KCLBasePlugin struct {
	h                     *resmap.PluginHelpers `json:"-"       yaml:"-"`
	functionConfiguration *yaml.RNode           `json:"-"       yaml:"-"`
	KCLPluginRun          `json:",inline" yaml:",inline"`
}

func (p *KCLBasePlugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	err := yaml.Unmarshal(c, &p.KCLPluginRun)
	if err != nil {
		return fmt.Errorf("while unmarshaling KCLRunnerConfig: %w", err)
	}
	// TODO: Should provide an alternative interface to configure from the functionConfig
	// RNode directly to avoid the unnecessary marshal and unmarshal.
	p.functionConfiguration, err = yaml.Parse(string(c))
	if err != nil {
		return fmt.Errorf("while parsing function configuration: %w", err)
	}
	return nil
}

func (p *KCLBasePlugin) ConfigureWithFunctionConfig(h *resmap.PluginHelpers, functionConfig *yaml.RNode) error {
	p.h = h
	p.functionConfiguration = functionConfig
	err := yaml.Unmarshal([]byte(functionConfig.MustString()), &p.KCLPluginRun)
	if err != nil {
		return fmt.Errorf("while decoding function configuration: %w", err)
	}
	return nil
}

func (p *KCLBasePlugin) loadParamResources() (resmap.ResMap, error) {
	result := resmap.New()
	for _, path := range p.ParamResources {
		resources, err := loadSource(p.h, path)
		if err != nil {
			return nil, fmt.Errorf("while loading param resource from path %s: %w", path, err)
		}
		err = result.AbsorbAll(resources)
		if err != nil {
			return nil, fmt.Errorf("while absorbing param resources from path %s: %w", path, err)
		}
	}
	return result, nil
}

func (p *KCLBasePlugin) setParamResourcesInFunctionConfig(paramResources resmap.ResMap) error {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	for _, r := range paramResources.ToRNodeSlice() {
		node.Content = append(node.Content, r.YNode())
	}
	seq := yaml.NewRNode(node)
	err := p.functionConfiguration.PipeE(
		yaml.Lookup("spec"),
		yaml.LookupCreate(yaml.MappingNode, "params"),
		yaml.SetField("resources", seq),
	)
	if err != nil {
		return fmt.Errorf("while setting param resources in function configuration: %w", err)
	}
	return nil
}

func (p *KCLBasePlugin) prepareFunctionConfig() error {
	if len(p.ParamResources) == 0 {
		return nil
	}
	paramResources, err := p.loadParamResources()
	if err != nil {
		return fmt.Errorf("while loading param resources: %w", err)
	}
	err = p.setParamResourcesInFunctionConfig(paramResources)
	if err != nil {
		return fmt.Errorf("while setting param resources in function configuration: %w", err)
	}
	return nil
}

var _ resmap.GeneratorPlugin = &KCLGeneratorPlugin{}

type KCLGeneratorPlugin struct {
	KCLBasePlugin `json:",inline" yaml:",inline"`
}

func (p *KCLGeneratorPlugin) Generate() (resmap.ResMap, error) {
	err := p.prepareFunctionConfig()
	if err != nil {
		return nil, fmt.Errorf("while preparing function configuration: %w", err)
	}
	var nodes []*yaml.RNode
	nodes, err = p.Transform(nil, p.functionConfiguration)
	if err != nil {
		return nil, fmt.Errorf("while transforming resources: %w", err)
	}
	var result resmap.ResMap
	result, err = p.h.ResmapFactory().NewResMapFromRNodeSlice(nodes)
	if err != nil {
		return nil, fmt.Errorf("while creating resmap from nodes: %w", err)
	}
	return result, nil
}

// NewKCLGeneratorPlugin returns a newly created KCLGenerator.
func NewKCLGeneratorPlugin() resmap.GeneratorPlugin {
	return &KCLGeneratorPlugin{}
}

var _ resmap.TransformerPlugin = &KCLTransformerPlugin{}

type KCLTransformerPlugin struct {
	KCLBasePlugin `json:",inline" yaml:",inline"`
}

func (p *KCLTransformerPlugin) Transform(m resmap.ResMap) error {
	err := p.prepareFunctionConfig()
	if err != nil {
		return fmt.Errorf("while preparing function configuration: %w", err)
	}
	var nodes []*yaml.RNode
	nodes, err = p.KCLPluginRun.Transform(m.ToRNodeSlice(), p.functionConfiguration)
	if err != nil {
		return fmt.Errorf("while transforming resources: %w", err)
	}
	var result resmap.ResMap
	result, err = p.h.ResmapFactory().NewResMapFromRNodeSlice(nodes)
	if err != nil {
		return fmt.Errorf("while creating resmap from nodes: %w", err)
	}
	err = m.AbsorbAll(result)
	if err != nil {
		return fmt.Errorf("while absorbing transformed resources: %w", err)
	}
	return nil
}

// NewKCLTransformerPlugin returns a newly created KCLTransformer.
func NewKCLTransformerPlugin() resmap.TransformerPlugin {
	return &KCLTransformerPlugin{}
}
