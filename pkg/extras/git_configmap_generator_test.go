package extras_test

// cSpell: words karmafun configmap filesys paralleltest storer worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

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

func TestGitConfigMapGeneratorPlugin_BadGitRepo(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	helpers, err := plugins.NewPluginHelpersInFileSystem(filesys.MakeFsInMemory())
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
	_, err = plugin.Generate()
	req.Error(err)
	req.ErrorContains(err, "opening git repo")
}

//nolint:paralleltest // This test changes the working directory, so can't be run in parallel with other tests
func TestGitConfigMapGeneratorPlugin_NoRemove(t *testing.T) {
	req := require.New(t)

	d := t.TempDir()
	gitRoot := filepath.Join(d, "repo")
	repo, err := git.PlainInitWithOptions(gitRoot, &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.Main,
		},
		Bare: false,
	})
	req.NoError(err)

	oldWd, err := os.Getwd()
	req.NoError(err)
	defer os.Chdir(oldWd) //nolint:errcheck // Best effort to restore working directory after test

	err = os.Chdir(gitRoot)
	req.NoError(err)

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err)

	plugin := extras.NewGitConfigMapGeneratorPlugin()
	config := []byte(`
apiVersion: karmafun.dev/v1alpha1
kind: GitConfigMapGenerator
metadata:
  name: git-info
files:
    - toto.yaml
`)
	err = plugin.Config(helpers, config)
	req.NoError(err)

	// Generate should work since we're running inside a git repo
	_, err = plugin.Generate()
	req.Error(err)
	req.ErrorContains(err, "getting remote origin")

	remote := "origin"
	_, err = repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: remote,
		URLs: []string{"https://example.com/repo.git"},
	})
	req.NoError(err)

	_, err = plugin.Generate()
	req.Error(err)
	req.ErrorContains(err, "getting current branch")

	wt, err := repo.Worktree()
	req.NoError(err)

	// Create an initial commit, otherwise repo.Head() will fail since there's no current branch
	_, err = wt.Commit("initial commit", &git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@karmafun.dev",
			When:  time.Now(),
		},
	})
	req.NoError(err)

	_, err = plugin.Generate()
	req.Error(err)
	req.ErrorContains(err, "creating config map from args")
}
