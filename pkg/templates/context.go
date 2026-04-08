package templates

// cSpell: words apimachinery filesys unmarshalling
import (
	"fmt"
	"log/slog"

	"github.com/getsops/sops/v3/cmd/sops/formats"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
)

const (
	defaultApiVersion = "config.karmafun.dev/v1alpha1"
	defaultKind       = "PlatformValues"
	defaultName       = "platform-values"
)

func MergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	// fill the out map with the first map
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v == nil {
			continue
		}
		if v, ok := v.(map[string]any); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]any); ok {
					// if b and out map has a map value, merge it too
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

type PlatformValues struct {
	Data                         map[string]any `json:"data"`
	metaV1.PartialObjectMetadata `               json:",inline"`
}

func NewPlatformValues() *PlatformValues {
	return &PlatformValues{
		PartialObjectMetadata: metaV1.PartialObjectMetadata{
			TypeMeta: metaV1.TypeMeta{
				APIVersion: defaultApiVersion,
				Kind:       defaultKind,
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name: defaultName,
			},
		},
		Data: make(map[string]any),
	}
}

func ReadPlatformValues(fs filesys.FileSystem, path string) (*PlatformValues, error) {
	if !fs.Exists(path) {
		slog.Debug("Platform values file does not exist, returning empty values", "path", path)
		// return an empty PlatformValues if the file doesn't exist, instead of an error.
		return NewPlatformValues(), nil
	}

	content, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading platform values from %s: %w", path, err)
	}
	var values PlatformValues
	err = yaml.Unmarshal(content, &values)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling platform values from %s: %w", path, err)
	}
	return &values, nil
}

func ReadSecretsValues(fs filesys.FileSystem, path string) (*PlatformValues, error) {
	if !fs.Exists(path) {
		slog.Debug("Secrets values file does not exist, returning empty values", "path", path)
		// return an empty PlatformValues if the file doesn't exist, instead of an error.
		return NewPlatformValues(), nil
	}
	format := formats.FormatForPath(path)

	content, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading secrets values from %s: %w", path, err)
	}

	decryptedContent, err := extras.Decrypt(content, format, formats.Yaml, false)
	if err != nil {
		return nil, fmt.Errorf("while decrypting secrets values from %s: %w", path, err)
	}

	var values PlatformValues
	err = yaml.Unmarshal(decryptedContent, &values)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling secrets values from %s: %w", path, err)
	}
	return &values, nil
}

func MergeValues(platformValues, secretsValues *PlatformValues) *PlatformValues {
	if platformValues == nil {
		slog.Debug("Platform values are nil, returning secrets values", "secretsValues", secretsValues)
		return secretsValues
	}

	if secretsValues == nil {
		slog.Debug("Secrets values are nil, returning platform values", "platformValues", platformValues)
		return platformValues
	}
	slog.Debug(
		"Merging platform values and secrets values",
		"platformValuesName",
		platformValues.Name,
		"secretsValuesName",
		secretsValues.Name,
	)
	merged := &PlatformValues{
		PartialObjectMetadata: platformValues.PartialObjectMetadata,
		Data:                  MergeMaps(platformValues.Data, secretsValues.Data),
	}
	return merged
}

func (p *PlatformValues) AsMap() map[string]any {
	return map[string]any{
		"data": p.Data,
	}
}
