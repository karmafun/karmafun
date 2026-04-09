package extras_test

// cSpell: words karmafun configmap

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestNewGitConfigMapGeneratorPlugin(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	req.NotNil(plugin)
}

func TestGitConfigMapGeneratorPlugin_Config(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: GitConfigMapGenerator
metadata:
  name: git-info
  namespace: default
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestGitConfigMapGeneratorPlugin_Config_WithRemoteName(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: GitConfigMapGenerator
metadata:
  name: git-info
remoteName: upstream
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)
}

func TestGitConfigMapGeneratorPlugin_Config_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	err = plugin.Config(helpers, []byte("{invalid yaml"))
	req.Error(err)
}

func TestGitConfigMapGeneratorPlugin_Generate(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: GitConfigMapGenerator
metadata:
  name: git-info
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	// Generate should work since we're running inside a git repo
	result, err := plugin.Generate()
	req.NoError(err)
	req.NotNil(result)
	req.Equal(1, result.Size())

	resources := result.Resources()
	req.Len(resources, 1)

	// Check the ConfigMap has the expected data
	repoURL, err := resources[0].GetFieldValue("data.repoURL")
	req.NoError(err)
	req.NotEmpty(repoURL)

	targetRevision, err := resources[0].GetFieldValue("data.targetRevision")
	req.NoError(err)
	req.NotEmpty(targetRevision)
}
