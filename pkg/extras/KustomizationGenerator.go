package extras

// cSpell: words filesys

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// KustomizationGeneratorPlugin configures the KustomizationGenerator.
type KustomizationGeneratorPlugin struct {
	Directory string `json:"kustomizeDirectory,omitempty" yaml:"kustomizeDirectory,omitempty"`
}

// enablePlugins adds to opts the options to run exec functions.
func enablePlugins(opts *krusty.Options) *krusty.Options {
	opts.PluginConfig = types.EnabledPluginConfig(types.BploUseStaticallyLinked) // cSpell: disable-line
	opts.PluginConfig.FnpLoadingOptions.EnableExec = true
	opts.PluginConfig.FnpLoadingOptions.AsCurrentUser = true
	opts.PluginConfig.HelmConfig.Command = "helm"
	opts.LoadRestrictions = types.LoadRestrictionsNone
	return opts
}

// runKustomizations runs the kustomization in dirname (URL compatible) with
// the filesystem fs.
func runKustomizations(fs filesys.FileSystem, dirname string) (resmap.ResMap, error) {
	opts := enablePlugins(krusty.MakeDefaultOptions())
	k := krusty.MakeKustomizer(opts)
	resources, err := k.Run(fs, dirname)
	if err != nil {
		return nil, fmt.Errorf("while running kustomizations in %s: %w", dirname, err)
	}
	return resources, nil
}

// Config reads the function configuration, i.e. the kustomizeDirectory.
func (p *KustomizationGeneratorPlugin) Config(_ *resmap.PluginHelpers, c []byte) error {
	err := yaml.Unmarshal(c, p)
	if err != nil {
		return fmt.Errorf("while configuring generator: %w", err)
	}
	return nil
}

// Generate generates the resources of the directory.
func (p *KustomizationGeneratorPlugin) Generate() (resmap.ResMap, error) {
	return runKustomizations(filesys.MakeFsOnDisk(), p.Directory)
}

// NewKustomizationGeneratorPlugin returns a newly Created KustomizationGenerator.
func NewKustomizationGeneratorPlugin() resmap.GeneratorPlugin {
	return &KustomizationGeneratorPlugin{}
}
