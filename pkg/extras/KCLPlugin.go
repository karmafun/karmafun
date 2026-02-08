package extras

// cSpell: words kcl

import (
	"fmt"

	"kcl-lang.io/krm-kcl/pkg/config"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KCLBasePlugin struct {
	h                     *resmap.PluginHelpers `json:"-"       yaml:"-"`
	functionConfiguration *yaml.RNode           `json:"-"       yaml:"-"`
	config.KCLRun         `json:",inline" yaml:",inline"`
}

func (p *KCLBasePlugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	err := yaml.Unmarshal(c, &p.KCLRun)
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
	err := yaml.Unmarshal([]byte(functionConfig.MustString()), &p.KCLRun)
	if err != nil {
		return fmt.Errorf("while decoding function configuration: %w", err)
	}
	return nil
}

var _ resmap.GeneratorPlugin = &KCLGeneratorPlugin{}

type KCLGeneratorPlugin struct {
	KCLBasePlugin `json:",inline" yaml:",inline"`
}

func (p *KCLGeneratorPlugin) Generate() (resmap.ResMap, error) {
	nodes, err := p.Transform(nil, p.functionConfiguration)
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
	nodes, err := p.KCLRun.Transform(m.ToRNodeSlice(), p.functionConfiguration)
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
