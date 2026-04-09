package extras_test

// cSpell: words karmafun kustomization

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/extras"
)

func TestNewKustomizationGeneratorPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKustomizationGeneratorPlugin()
	req.NotNil(plugin)
}

func TestKustomizationGeneratorPlugin_Config(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKustomizationGeneratorPlugin()
	config := []byte(`
kustomizeDirectory: /some/directory
`)
	err := plugin.Config(nil, config)
	req.NoError(err)
}

func TestKustomizationGeneratorPlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKustomizationGeneratorPlugin()
	err := plugin.Config(nil, []byte("{invalid yaml"))
	req.Error(err)
}

func TestKustomizationGeneratorPlugin_Config_EmptyConfig(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewKustomizationGeneratorPlugin()
	err := plugin.Config(nil, []byte(`{}`))
	req.NoError(err)
}

func TestRunKustomizations_InvalidDirectory(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Use the disk filesystem but with a nonexistent path - should return an error
	_, err := extras.RunKustomizations(filesys.MakeFsOnDisk(), "/tmp/this-dir-does-not-exist-karmafun-test")
	req.Error(err)
}

func TestKustomizationGeneratorPlugin_Generate_SimpleKustomization(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Create a temporary directory with a simple kustomization
	tmpDir, err := os.MkdirTemp("", "karmafun-kust-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Create a kustomization.yaml that generates a simple configmap
	kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
  - name: test-cm
    literals:
      - key=value
`
	err = os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte(kustomizationContent), 0600)
	req.NoError(err)

	plugin := extras.NewKustomizationGeneratorPlugin()
	config := []byte("kustomizeDirectory: " + tmpDir)
	err = plugin.Config(nil, config)
	req.NoError(err)

	result, err := plugin.Generate()
	req.NoError(err)
	req.NotNil(result)
	req.GreaterOrEqual(result.Size(), 1)
}
