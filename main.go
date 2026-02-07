package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/plugins"
	"github.com/karmafun/karmafun/pkg/utils"
)

var (
	KarmafunVersion = "v0.4.3" // <---VERSION--->
	Commit          = "unknown"
	BuildDate       = "unknown"
	BuiltBy         = "unknown"
)

var _ framework.ResourceListProcessor = &processor{}

type processor struct{}

func appendResources(rl *framework.ResourceList, rrl []*yaml.RNode, annotations map[string]string) error {
	if err := utils.TransferAnnotations(rrl, annotations); err != nil {
		return fmt.Errorf("while transferring annotations: %w", err)
	}

	rl.Items = append(rl.Items, rrl...)
	return nil
}

func transformResources(
	rl *framework.ResourceList,
	transformer resmap.Transformer,
	annotations map[string]string,
) error {
	rm := utils.ResourceMapFromNodes(rl.Items)
	err := transformer.Transform(rm)
	if err != nil {
		return fmt.Errorf("transforming resources: %w", err)
	}

	if _, ok := annotations[utils.FunctionAnnotationCleanup]; ok {
		for _, r := range rm.Resources() {
			utils.RemoveBuildAnnotations(r)
		}
	}

	rl.Items = rm.ToRNodeSlice()

	// If the annotation `config.karmafun.dev/prune-local` is present in a
	// transformer makes all the local resources disappear.
	if _, ok := annotations[utils.FunctionAnnotationPruneLocal]; ok {
		err = rl.Filter(utils.UnLocal)
		if err != nil {
			return fmt.Errorf("while pruning `config.karmafun.dev/local-config` resources: %w", err)
		}
	}
	return nil
}

func generateResources(rl *framework.ResourceList, generator resmap.Generator, annotations map[string]string) error {
	rm, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("generating resource(s): %w", err)
	}

	rrl := rm.ToRNodeSlice()

	if err = appendResources(rl, rrl, annotations); err != nil {
		return fmt.Errorf("while appending generated resources: %w", err)
	}
	return nil
}

func (p *processor) Process(rl *framework.ResourceList) error {
	config := rl.FunctionConfig
	configAnnotations := config.GetAnnotations()

	res := resource.Resource{RNode: *config}

	plugin, err := plugins.MakeBuiltinPlugin(resid.GvkFromNode(config))
	if err != nil || plugin == nil {
		// Check if config asks us to inject it
		if _, ok := configAnnotations[utils.FunctionAnnotationInjectLocal]; !ok {
			return fmt.Errorf("creating plugin: %w", err)
		}
		// In this case there is no plugin, but we will inject the config as a resource in the list,
		// so we can still process it
		if err = appendResources(rl, []*yaml.RNode{config}, configAnnotations); err != nil {
			return fmt.Errorf("while injecting local config: %w", err)
		}
		return nil
	}

	yamlNode := config.YNode()
	var yamlBytes []byte
	yamlBytes, err = yaml.Marshal(yamlNode)
	if err != nil {
		return fmt.Errorf("marshaling yaml from res %s: %w", res.OrgId(), err)
	}
	helpers, err := plugins.NewPluginHelpers()
	if err != nil {
		return fmt.Errorf("cannot build Plugin helpers: %w", err)
	}
	err = plugin.Config(helpers, yamlBytes)
	if err != nil {
		return fmt.Errorf("plugin %s fails configuration: %w", res.OrgId(), err)
	}

	switch v := plugin.(type) {
	case resmap.Transformer:
		if err = transformResources(rl, v, configAnnotations); err != nil {
			return fmt.Errorf("while transforming resources: %w", err)
		}
	case resmap.Generator:
		if err = generateResources(rl, v, configAnnotations); err != nil {
			return fmt.Errorf("while generating resources: %w", err)
		}
	default:
		return fmt.Errorf("plugin %s is neither a generator nor a transformer", res.OrgId())
	}

	return nil
}

func main() {
	cmd := command.Build(&processor{}, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	cmd.Version = KarmafunVersion

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
