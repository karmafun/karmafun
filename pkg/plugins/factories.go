package plugins

// cSpell: words filesys restrictor pldr gosec govet konfig
import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	//nolint:staticcheck // mimics the kustomize pattern used for plugins.
	"sigs.k8s.io/kustomize/api/builtins"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/konfig"
	fLdr "sigs.k8s.io/kustomize/api/pkg/loader"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/utils"
)

type FunctionConfigConfigurable interface {
	ConfigureWithFunctionConfig(h *resmap.PluginHelpers, functionConfig *yaml.RNode) error
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=BuiltinPluginType
type BuiltinPluginType int

const (
	Unknown BuiltinPluginType = iota
	AnnotationsTransformer
	ConfigMapGenerator
	IAMPolicyGenerator
	HashTransformer
	ImageTagTransformer
	LabelTransformer
	NamespaceTransformer
	PatchJson6902Transformer
	PatchStrategicMergeTransformer
	PatchTransformer
	PrefixSuffixTransformer
	PrefixTransformer
	SuffixTransformer
	ReplicaCountTransformer
	SecretGenerator
	ValueAddTransformer
	HelmChartInflationGenerator
	ReplacementTransformer
	GitConfigMapGenerator
	RemoveTransformer
	KustomizationGenerator
	SopsGenerator
	KCLGenerator
	KCLTransformer
)

var stringToBuiltinPluginTypeMap map[string]BuiltinPluginType

func init() { //nolint:gochecknoinits // kustomize pattern used for plugins.
	stringToBuiltinPluginTypeMap = makeStringToBuiltinPluginTypeMap()
}

func makeStringToBuiltinPluginTypeMap() map[string]BuiltinPluginType {
	result := make(map[string]BuiltinPluginType, 25)
	for k := range TransformerFactories {
		result[k.String()] = k
	}
	for k := range GeneratorFactories {
		result[k.String()] = k
	}
	return result
}

func GetBuiltinPluginType(n string) BuiltinPluginType {
	result, ok := stringToBuiltinPluginTypeMap[n]
	if ok {
		return result
	}
	return Unknown
}

type MultiTransformer struct {
	transformers []resmap.TransformerPlugin
}

func (t *MultiTransformer) Transform(m resmap.ResMap) error {
	for _, transformer := range t.transformers {
		if err := transformer.Transform(m); err != nil {
			return fmt.Errorf("transforming resources: %w", err)
		}
	}
	return nil
}

func (t *MultiTransformer) Config(h *resmap.PluginHelpers, b []byte) error {
	for _, transformer := range t.transformers {
		if err := transformer.Config(h, b); err != nil {
			return fmt.Errorf("configuring transformer: %w", err)
		}
	}
	return nil
}

func NewMultiTransformer() resmap.TransformerPlugin {
	return &MultiTransformer{[]resmap.TransformerPlugin{
		builtins.NewPrefixTransformerPlugin(),
		builtins.NewSuffixTransformerPlugin(),
	}}
}

var TransformerFactories = map[BuiltinPluginType]func() resmap.TransformerPlugin{
	AnnotationsTransformer:         builtins.NewAnnotationsTransformerPlugin,
	HashTransformer:                builtins.NewHashTransformerPlugin,
	ImageTagTransformer:            builtins.NewImageTagTransformerPlugin,
	LabelTransformer:               builtins.NewLabelTransformerPlugin,
	NamespaceTransformer:           builtins.NewNamespaceTransformerPlugin,
	PatchJson6902Transformer:       builtins.NewPatchJson6902TransformerPlugin,
	PatchStrategicMergeTransformer: builtins.NewPatchStrategicMergeTransformerPlugin,
	PatchTransformer:               builtins.NewPatchTransformerPlugin,
	PrefixSuffixTransformer:        NewMultiTransformer,
	PrefixTransformer:              builtins.NewPrefixTransformerPlugin,
	SuffixTransformer:              builtins.NewSuffixTransformerPlugin,
	ReplacementTransformer:         extras.NewExtendedReplacementTransformerPlugin,
	ReplicaCountTransformer:        builtins.NewReplicaCountTransformerPlugin,
	ValueAddTransformer:            builtins.NewValueAddTransformerPlugin,
	RemoveTransformer:              extras.NewRemoveTransformerPlugin,
	KCLTransformer:                 extras.NewKCLTransformerPlugin,
	// Do not wired SortOrderTransformer as a builtin plugin.
	// We only want it to be available in the top-level kustomization.
	// See: https://github.com/kubernetes-sigs/kustomize/issues/3913
}

var GeneratorFactories = map[BuiltinPluginType]func() resmap.GeneratorPlugin{
	ConfigMapGenerator:          builtins.NewConfigMapGeneratorPlugin,
	IAMPolicyGenerator:          builtins.NewIAMPolicyGeneratorPlugin,
	SecretGenerator:             builtins.NewSecretGeneratorPlugin,
	HelmChartInflationGenerator: builtins.NewHelmChartInflationGeneratorPlugin,
	GitConfigMapGenerator:       extras.NewGitConfigMapGeneratorPlugin,
	KustomizationGenerator:      extras.NewKustomizationGeneratorPlugin,
	SopsGenerator:               extras.NewSopsGeneratorPlugin,
	KCLGenerator:                extras.NewKCLGeneratorPlugin,
}

func MakeBuiltinPlugin(r resid.Gvk) (resmap.Configurable, error) {
	bpt := GetBuiltinPluginType(r.Kind)
	if f, ok := TransformerFactories[bpt]; ok {
		return f(), nil
	}
	if f, ok := GeneratorFactories[bpt]; ok {
		return f(), nil
	}
	return nil, fmt.Errorf("unable to load builtin %s", r)
}

func RestrictionNone(
	_ filesys.FileSystem, _ filesys.ConfirmedDir, path string,
) (string, error) {
	return path, nil
}

type LoadRestrictorFunc func(
	filesys.FileSystem, filesys.ConfirmedDir, string) (string, error)

//nolint:govet // Data structure alignment matches kustomize's internal loader.
type FileLoader struct {
	referrer       *FileLoader
	root           filesys.ConfirmedDir
	loadRestrictor LoadRestrictorFunc
}

func NewPluginHelpers() (*resmap.PluginHelpers, error) {
	depProvider := provider.NewDepProvider()

	fSys := filesys.MakeFsOnDisk()
	resmapFactory := resmap.NewFactory(depProvider.GetResourceFactory())
	resmapFactory.RF().IncludeLocalConfigs = true

	// The loader creation methods have gone internal in kustomize.
	// We have to create a loader and then use unsafe to modify the load restrictor to allow loading from any path.
	// This is brittle but there doesn't seem to be a better option without copying the loader code into our plugin.
	var ldr ifc.Loader
	pldr := fLdr.NewFileLoaderAtCwd(fSys)
	ptr := unsafe.Pointer(pldr) //nolint:gosec // we need to use unsafe to modify the load restrictor.
	unrestricted := (*FileLoader)(ptr)
	unrestricted.loadRestrictor = RestrictionNone
	ldr = pldr

	config := types.DisabledPluginConfig()
	config.HelmConfig.Enabled = true
	config.HelmConfig.Command = "helm"
	return resmap.NewPluginHelpers(ldr, depProvider.GetFieldValidator(), resmapFactory, config), nil
}

func GetPluginPath() (string, error) {
	baseDir, ok := os.LookupEnv(konfig.KustomizePluginHomeEnv)
	if !ok {
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("while getting user config dir: %w", err)
		}
		baseDir = filepath.Join(baseDir, konfig.ProgramName, konfig.RelPluginHome)
	}

	baseDir = filepath.Join(baseDir, "karmafun.dev", "v1alpha1")
	return baseDir, nil
}

func CreatePluginDirectoryHierarchy(fs filesys.FileSystem) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("while getting executable path: %w", err)
	}
	baseDir, err := GetPluginPath()
	if err != nil {
		return fmt.Errorf("while getting plugin path: %w", err)
	}
	slog.Debug("Creating symlinks for plugins...", "path", baseDir)
	err = fs.RemoveAll(baseDir)
	if err != nil {
		return fmt.Errorf("while removing existing plugin directory hierarchy at %s: %w", baseDir, err)
	}
	err = fs.MkdirAll(baseDir)
	if err != nil {
		return fmt.Errorf("while creating plugin directory hierarchy as %s: %w", baseDir, err)
	}

	execConfirmedDir := filesys.ConfirmedDir(filepath.Dir(execPath))

	// Loop through all the builtin plugins and create symlinks to the executable in the plugin directory hierarchy.
	for pluginType := range utils.Concat(maps.Keys(TransformerFactories), maps.Keys(GeneratorFactories)) {
		kind := pluginType.String()
		lowerKind := strings.ToLower(kind)
		pluginPath := filepath.Join(baseDir, lowerKind, kind)
		err = fs.MkdirAll(filepath.Dir(pluginPath))
		if err != nil {
			return fmt.Errorf("while creating plugin directory for %s: %w", kind, err)
		}
		// Create a symlink to the executable for the plugin if it doesn't already exist.
		if fs.Exists(pluginPath) {
			continue
		}
		if err := os.Symlink(execConfirmedDir.Join(filepath.Base(execPath)), pluginPath); err != nil {
			return fmt.Errorf("while creating symlink for plugin %s at %s: %w", kind, pluginPath, err)
		}
	}

	return nil
}

func RemovePluginDirectoryHierarchy(fs filesys.FileSystem) error {
	baseDir, err := GetPluginPath()
	if err != nil {
		return fmt.Errorf("while getting plugin path: %w", err)
	}
	// remove version
	baseDir = filepath.Dir(baseDir)
	if fs.Exists(baseDir) {
		slog.Debug("Removing plugin directory hierarchy at path", "path", baseDir)
		if err := fs.RemoveAll(baseDir); err != nil {
			return fmt.Errorf("while removing plugin directory hierarchy at %s: %w", baseDir, err)
		}
	} else {
		slog.Debug("Plugin directory hierarchy does not exist, skipping removal", "path", baseDir)
	}
	return nil
}
