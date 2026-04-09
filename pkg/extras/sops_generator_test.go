package extras_test

// cSpell: words karmafun sops agekey

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestNewSopsGeneratorPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewSopsGeneratorPlugin()
	req.NotNil(plugin)
}

func TestSopsGeneratorPlugin_Config_WithFiles(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewSopsGeneratorPlugin()
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
  name: test-sops
files:
  - secrets.sops.yaml
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestSopsGeneratorPlugin_Config_WithSopsInline(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewSopsGeneratorPlugin()
	// The "sops" json field being non-nil triggers the buffer mode
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
  name: test-sops
sops:
  version: "3.0.0"
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestSopsGeneratorPlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewSopsGeneratorPlugin()
	err = plugin.Config(helpers, []byte("{invalid yaml"))
	req.Error(err)
}

func TestSopsGeneratorPlugin_Config_NoFilesOrSops(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewSopsGeneratorPlugin()
	// No files and no sops should error
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
  name: test-sops
`)
	err = plugin.Config(helpers, config)
	req.Error(err)
	req.Contains(err.Error(), "file")
}

func TestDecrypt_InvalidInput(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	_, err := extras.Decrypt([]byte("not encrypted"), 0, 0, false)
	req.Error(err)
}

func TestDecryptToRNodes_InvalidInput(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	_, err := extras.DecryptToRNodes([]byte("not encrypted"), 0, false)
	req.Error(err)
}

func TestSopsGeneratorPlugin_Generate_WithSopsBuffer(t *testing.T) {
	// This test requires the SOPS_AGE_KEY to be set to test actual decryption.
	// We test that calling Generate with an inline (buffer) config that is not
	// actually encrypted returns an error (since we have no valid key).
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewSopsGeneratorPlugin()
	// This is not actually SOPS-encrypted content, so Generate should fail
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
  name: test-sops
sops:
  version: "3.0.0"
data:
  key: plaintext-not-encrypted
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	_, err = plugin.Generate()
	req.Error(err)
}
