package plugins_test

// cSpell: words filesys testdir

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestNewPluginHelpers_CheckLoader(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	fSys := filesys.MakeFsOnDisk()
	// Create temporary directories and files for testing the loader. These will be cleaned up at the end of the test.
	tempDir, err := os.MkdirTemp("", "karmafun-test-")
	req.NoError(err, "creating temporary directory should not error")
	defer func() {
		err = os.RemoveAll(tempDir)
		req.NoError(err, "removing temporary directory should not error")
	}()

	err = fSys.Mkdir(tempDir + "/testdir")
	req.NoError(err, "creating test directory should not error")
	err = fSys.Mkdir(tempDir + "/testdir2")
	req.NoError(err, "creating test directory should not error")
	// Create a file in testdir2 to ensure that the loader can load files from outside the current directory.
	err = fSys.WriteFile(tempDir+"/testdir2/testfile.txt", []byte("test"))
	req.NoError(err, "creating test file should not error")
	// set current directory to the test directory to ensure that the loader is rooted at the current directory.
	err = os.Chdir(tempDir + "/testdir")
	req.NoError(err, "changing current directory should not error")

	helpers, err := plugins.NewPluginHelpers()
	req.NoError(err, "creating plugin helpers should not error")
	loader := helpers.Loader()
	// The loader should allow loading from any path, not just the current directory.
	// This is necessary for plugins that need to load files from outside the current directory, such as the SopsGenerator.
	newLoader, err := loader.New("../testdir2")
	req.NoError(err, "loader should allow loading from any path")
	// Load test file to ensure that the loader is actually working.
	_, err = newLoader.Load("testfile.txt")
	req.NoError(err, "loader should be able to load files from any path")
	// Load the test file from the original loader to ensure that it is also working.
	_, err = loader.Load("../testdir2/testfile.txt")
	req.NoError(err, "original loader should also be able to load files from any path")
}
