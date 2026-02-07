package extras

// cSpell: words oyaml keyservice

import (
	"fmt"

	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/keyservice"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/kio"
	yaml "sigs.k8s.io/kustomize/kyaml/yaml"
	oyaml "sigs.k8s.io/yaml"

	"github.com/karmafun/karmafun/pkg/utils"
)

const (
	defaultApiVersion = "config.karmafun.dev/v1alpha1"
	defaultKind       = "PlatformSecrets"
)

// SopsGeneratorPlugin configures the SopsGenerator.
type SopsGeneratorPlugin struct {
	yaml.ResourceMeta

	Files []string `yaml:"files,omitempty"`

	Sops map[string]any `json:"sops,omitempty" yaml:"spec,omitempty"`

	h      *resmap.PluginHelpers
	buffer []byte
}

func Decrypt(b []byte, format formats.Format, file string, ignoreMac bool) ([]*yaml.RNode, error) {
	store := common.StoreForFormat(format)

	// Load SOPS file and access the data key
	tree, err := store.LoadEncryptedFile(b)
	if err != nil {
		return nil, fmt.Errorf("while loading encrypted file %s: %w", file, err)
	}

	_, err = common.DecryptTree(common.DecryptTreeOpts{
		KeyServices: []keyservice.KeyServiceClient{
			keyservice.NewLocalClient(),
		},
		Tree:      &tree,
		IgnoreMac: ignoreMac,
		Cipher:    aes.NewCipher(),
	})
	if err != nil {
		return nil, fmt.Errorf("while decrypting tree for file %s: %w", file, err)
	}

	var data []byte

	data, err = store.EmitPlainFile(tree.Branches)
	if err != nil {
		return nil, fmt.Errorf("trouble decrypting file %s: %w", file, err)
	}

	var nodes []*yaml.RNode
	nodes, err = kio.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("while reading decrypted resources from file %s: %w", file, err)
	}
	return nodes, nil
}

// Config reads the function configuration, i.e. the kustomizeDirectory.
func (p *SopsGeneratorPlugin) Config(h *resmap.PluginHelpers, c []byte) error {
	err := oyaml.Unmarshal(c, p)
	if err != nil {
		return fmt.Errorf("while unmarshaling configuration: %w", err)
	}
	p.h = h
	if p.Sops != nil {
		p.buffer = c
	} else if p.Files == nil {
		return fmt.Errorf("generator configuration doesn't contain any file")
	}
	return nil
}

func decryptBuffer(buffer []byte, name string, format formats.Format) ([]*yaml.RNode, error) {
	if buffer == nil {
		return nil, fmt.Errorf("buffer is nil for manifest %q", name)
	}
	nodes, err := Decrypt(buffer, format, name, true)
	if err != nil {
		return nil, fmt.Errorf("error decoding manifest %q, content -->%s<--: %w", name, string(buffer), err)
	}

	for _, r := range nodes {
		r.SetKind(defaultKind)
		r.SetApiVersion(defaultApiVersion)

		if err := r.PipeE(yaml.SetAnnotation(utils.FunctionAnnotationInjectLocal, "true")); err != nil {
			return nil, fmt.Errorf("while setting annotation on resource from file %s: %w", name, err)
		}
	}

	return nodes, nil
}

func decryptFiles(files []string, loader ifc.Loader) ([]*yaml.RNode, error) {
	var nodes []*yaml.RNode
	for _, file := range files {
		b, err := loader.Load(file)
		if err != nil {
			return nil, fmt.Errorf("while reading manifest %q: %w", file, err)
		}

		format := formats.FormatForPath(file)
		fileNodes, err := Decrypt(b, format, file, false)
		if err != nil {
			return nil, fmt.Errorf("while decrypting file %q: %w", file, err)
		}
		nodes = append(nodes, fileNodes...)
	}
	return nodes, nil
}

// Generate generates the resources of the directory.
func (p *SopsGeneratorPlugin) Generate() (resmap.ResMap, error) {
	var nodes []*yaml.RNode
	var err error
	if p.buffer != nil {
		nodes, err = decryptBuffer(p.buffer, p.GetIdentifier().Name, formats.Yaml)
		if err != nil {
			return nil, fmt.Errorf("error decrypting buffer: %w", err)
		}
	} else {
		nodes, err = decryptFiles(p.Files, p.h.Loader())
		if err != nil {
			return nil, fmt.Errorf("error decrypting files: %w", err)
		}
	}
	return utils.ResourceMapFromNodes(nodes), nil
}

// NewSopsGeneratorPlugin returns a newly Created SopsGenerator.
func NewSopsGeneratorPlugin() resmap.GeneratorPlugin {
	return &SopsGeneratorPlugin{}
}
