package plugins_test

// cSpell: words filesys testdir karmafun

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"

	"github.com/karmafun/karmafun/pkg/plugins"
)

func TestGetBuiltinPluginType_Known(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind string
		want plugins.BuiltinPluginType
	}{
		{"AnnotationsTransformer", "AnnotationsTransformer", plugins.AnnotationsTransformer},
		{"ConfigMapGenerator", "ConfigMapGenerator", plugins.ConfigMapGenerator},
		{"SecretGenerator", "SecretGenerator", plugins.SecretGenerator},
		{"ReplacementTransformer", "ReplacementTransformer", plugins.ReplacementTransformer},
		{"GitConfigMapGenerator", "GitConfigMapGenerator", plugins.GitConfigMapGenerator},
		{"RemoveTransformer", "RemoveTransformer", plugins.RemoveTransformer},
		{"KustomizationGenerator", "KustomizationGenerator", plugins.KustomizationGenerator},
		{"SopsGenerator", "SopsGenerator", plugins.SopsGenerator},
		{"LabelTransformer", "LabelTransformer", plugins.LabelTransformer},
		{"NamespaceTransformer", "NamespaceTransformer", plugins.NamespaceTransformer},
		{"ImageTagTransformer", "ImageTagTransformer", plugins.ImageTagTransformer},
		{"PrefixTransformer", "PrefixTransformer", plugins.PrefixTransformer},
		{"SuffixTransformer", "SuffixTransformer", plugins.SuffixTransformer},
		{"KCLGenerator", "KCLGenerator", plugins.KCLGenerator},
		{"KCLTransformer", "KCLTransformer", plugins.KCLTransformer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)
			got := plugins.GetBuiltinPluginType(tt.kind)
			req.Equal(tt.want, got)
		})
	}
}

func TestGetBuiltinPluginType_Unknown(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	got := plugins.GetBuiltinPluginType("NonExistentPlugin")
	req.Equal(plugins.Unknown, got)
}

func TestGetBuiltinPluginType_Empty(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	got := plugins.GetBuiltinPluginType("")
	req.Equal(plugins.Unknown, got)
}

func TestMakeBuiltinPlugin_Transformer(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "AnnotationsTransformer"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)
}

func TestMakeBuiltinPlugin_Generator(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "ConfigMapGenerator"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)
}

func TestMakeBuiltinPlugin_Unknown(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "UnknownPlugin"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.Error(err)
	req.Nil(plugin)
	req.Contains(err.Error(), "unable to load builtin")
}

func TestMakeBuiltinPlugin_ReplacementTransformer(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "ReplacementTransformer"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)
}

func TestMakeBuiltinPlugin_RemoveTransformer(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "RemoveTransformer"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)
}

func TestNewMultiTransformer(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "PrefixSuffixTransformer"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)

	h, err := plugins.NewPluginHelpers()
	req.NoError(err)
	// A valid config for PrefixSuffixTransformer requires fieldSpecs
	err = plugin.Config(h, []byte(`
prefix: my-
fieldSpecs:
  - path: metadata/name
`))
	req.NoError(err)
}

func TestMultiTransformer_Transform(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	gvk := resid.Gvk{Kind: "PrefixSuffixTransformer"}
	plugin, err := plugins.MakeBuiltinPlugin(gvk)
	req.NoError(err)
	req.NotNil(plugin)

	h, err := plugins.NewPluginHelpers()
	req.NoError(err)
	err = plugin.Config(h, []byte(`
prefix: my-
fieldSpecs:
  - path: metadata/name
`))
	req.NoError(err)

	// Cast to TransformerPlugin
	transformer, ok := plugin.(interface{ Transform(resmap.ResMap) error })
	req.True(ok)

	rm := resmap.New()
	rf := provider.NewDefaultDepProvider().GetResourceFactory()
	resources, err := rf.SliceFromBytes([]byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
`))
	req.NoError(err)
	for _, r := range resources {
		req.NoError(rm.Append(r))
	}

	err = transformer.Transform(rm)
	req.NoError(err)

	resourcesList := rm.Resources()
	req.Len(resourcesList, 1)
	req.Equal("my-test-cm", resourcesList[0].GetName())
}

func TestGetPluginPath_WithEnvVar(t *testing.T) {
	req := require.New(t)

	tmpDir, err := os.MkdirTemp("", "karmafun-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir)

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	path, err := plugins.GetPluginPath()
	req.NoError(err)
	req.NotEmpty(path)
	req.Contains(path, "karmafun.dev")
}

func TestGetPluginPath_WithoutEnvVar(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	path, err := plugins.GetPluginPath()
	req.NoError(err)
	req.NotEmpty(path)
	req.Contains(path, "karmafun.dev")
}

func TestRemovePluginDirectoryHierarchy_NotExist(t *testing.T) {
	req := require.New(t)

	tmpDir, err := os.MkdirTemp("", "karmafun-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir)

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	fs := filesys.MakeFsOnDisk()
	// Should not error when directory doesn't exist
	err = plugins.RemovePluginDirectoryHierarchy(fs)
	req.NoError(err)
}

func TestCreateAndRemovePluginDirectoryHierarchy(t *testing.T) {
	req := require.New(t)

	tmpDir, err := os.MkdirTemp("", "karmafun-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir)

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	fs := filesys.MakeFsOnDisk()

	err = plugins.CreatePluginDirectoryHierarchy(fs)
	req.NoError(err)

	// Verify the directory was created
	path, err := plugins.GetPluginPath()
	req.NoError(err)
	req.True(fs.Exists(path))

	// Now remove it
	err = plugins.RemovePluginDirectoryHierarchy(fs)
	req.NoError(err)
}

func TestRestrictionNone(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	fs := filesys.MakeFsOnDisk()
	confirmedDir := filesys.ConfirmedDir("/tmp")
	result, err := plugins.RestrictionNone(fs, confirmedDir, "/some/path")
	req.NoError(err)
	req.Equal("/some/path", result)
}

func TestNewPluginHelpers_CheckLoader(t *testing.T) {
	// Note: Not parallel since this test changes the process working directory.
	req := require.New(t)

	fSys := filesys.MakeFsOnDisk()
	// Create temporary directories and files for testing the loader. These will be cleaned up at the end of the test.
	tempDir, err := os.MkdirTemp("", "karmafun-test-")
	req.NoError(err, "creating temporary directory should not error")

	originalDir, err := os.Getwd()
	req.NoError(err, "getting current directory should not error")
	defer func() {
		// Restore original directory before cleanup so the directory still exists when we leave.
		err = os.Chdir(originalDir)
		req.NoError(err, "restoring original directory should not error")
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
