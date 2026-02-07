package extras

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type RemoveTransformerPlugin struct {
	Targets []*types.Selector `json:"targets,omitempty" yaml:"targets,omitempty"`
}

func (p *RemoveTransformerPlugin) Config(_ *resmap.PluginHelpers, c []byte) error {
	err := yaml.Unmarshal(c, p)
	if err != nil {
		return fmt.Errorf("while configuring RemoveTransformerPlugin: %w", err)
	}
	return nil
}

func (p *RemoveTransformerPlugin) Transform(m resmap.ResMap) error {
	if p.Targets == nil {
		return fmt.Errorf("must specify at least one target")
	}
	for _, t := range p.Targets {
		resources, err := m.Select(*t)
		if err != nil {
			return fmt.Errorf("while selecting target %s: %w", t.String(), err)
		}
		for _, r := range resources {
			err = m.Remove(r.CurId())
			if err != nil {
				return fmt.Errorf("while removing resource %s: %w", r.CurId().String(), err)
			}
		}
	}
	return nil
}

func NewRemoveTransformerPlugin() resmap.TransformerPlugin {
	return &RemoveTransformerPlugin{}
}
