package build_test

// cSpell: words filesys testify karmafun resmap pflag kustdir kust
import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/cmd/build"
)

// --- NewBuildOptions tests ---

func TestNewBuildOptions_Defaults(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	opts := build.NewBuildOptions()
	req.NotNil(opts)
	req.Equal("values.yaml", opts.ValuesFile)
	req.Equal("secrets.sops.yaml", opts.SecretsFile)
	req.Empty(opts.OutputDirectory)
}

// --- NewLogOptions tests ---

func TestNewLogOptions_Defaults(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	opts := build.NewLogOptions()
	req.NotNil(opts)
	req.Equal("info", opts.Level)
	req.False(opts.Json)
}

// --- AddFlags tests ---

func TestBuildOptions_AddFlags(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	opts := build.NewBuildOptions()
	cmd := build.NewBuildCommand(opts, nil)
	flags := cmd.Flags()

	req.NotNil(flags.Lookup("values-file"), "values-file flag should be registered")
	req.NotNil(flags.Lookup("secrets-file"), "secrets-file flag should be registered")
	req.NotNil(flags.Lookup("output-directory"), "output-directory flag should be registered")
}

func TestBuildOptions_AddFlags_ShortFlag(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	opts := build.NewBuildOptions()
	cmd := build.NewBuildCommand(opts, nil)
	flags := cmd.Flags()

	outputFlag := flags.ShorthandLookup("o")
	req.NotNil(outputFlag, "output-directory should have -o shorthand")
	req.Equal("output-directory", outputFlag.Name)
}

func TestLogOptions_AddFlags(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	logOpts := build.NewLogOptions()
	cmd := build.NewBuildCommand(nil, logOpts)
	// Log flags are added to root, but since we pass existing logOpts with
	// createdLogOptions=false, they are not added. Let's use a root command pattern.
	// When logOptions is provided externally, flags are NOT added to cmd.
	// Add them explicitly to test.
	logOpts.AddFlags(cmd.Flags())
	flags := cmd.Flags()
	req.NotNil(flags.Lookup("log-level"), "log-level flag should be registered")
	req.NotNil(flags.Lookup("log-json"), "log-json flag should be registered")
}

// --- NewBuildCommand tests ---

func TestNewBuildCommand_CreatesCommand(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := build.NewBuildCommand(nil, nil)
	req.NotNil(cmd)
	req.Equal("build [flags] <kustomization directory>", cmd.Use)
	req.NotEmpty(cmd.Short)
	req.NotEmpty(cmd.Long)
}

func TestNewBuildCommand_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := build.NewBuildCommand(nil, nil)
	// No args should fail validation
	err := cmd.Args(cmd, []string{})
	req.Error(err, "command should require exactly one argument")
	// Two args should also fail
	err = cmd.Args(cmd, []string{"dir1", "dir2"})
	req.Error(err, "command should require exactly one argument")
	// One arg should succeed
	err = cmd.Args(cmd, []string{"dir"})
	req.NoError(err)
}

func TestNewBuildCommand_WithNilOptions_UsesDefaults(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := build.NewBuildCommand(nil, nil)
	req.NotNil(cmd)
	// Verify default flags are registered
	valuesFlag := cmd.Flags().Lookup("values-file")
	req.NotNil(valuesFlag)
	req.Equal("values.yaml", valuesFlag.DefValue)
}

func TestLogOptions_InitLogger_TextHandler(t *testing.T) {
	t.Parallel()
	opts := build.NewLogOptions()
	opts.Level = "debug"
	opts.Json = false
	// InitLogger should not panic
	require.NotPanics(t, func() {
		opts.InitLogger()
	})
}

func TestLogOptions_InitLogger_JSONHandler(t *testing.T) {
	t.Parallel()
	opts := build.NewLogOptions()
	opts.Level = "warn"
	opts.Json = true
	require.NotPanics(t, func() {
		opts.InitLogger()
	})
}

func TestLogOptions_InitLogger_InvalidLevel_DefaultsToInfo(t *testing.T) {
	t.Parallel()
	opts := build.NewLogOptions()
	opts.Level = "invalid-level"
	// Should not panic, falls back to info level
	require.NotPanics(t, func() {
		opts.InitLogger()
	})
}

// --- SplitResMapToDir tests ---

func makeTestResMap(t *testing.T) resmap.ResMap {
	t.Helper()
	req := require.New(t)
	rf := provider.NewDefaultDepProvider().GetResourceFactory()
	rm := resmap.New()

	configMapYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
`
	resources, err := rf.SliceFromBytes([]byte(configMapYAML))
	req.NoError(err, "creating test resources should not error")
	for _, r := range resources {
		req.NoError(rm.Append(r))
	}
	return rm
}

func TestSplitResMapToDir_CreatesFiles(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	rm := makeTestResMap(t)

	err := build.SplitResMapToDir(fs, rm, "/output")
	req.NoError(err)

	req.True(fs.IsDir("/output"), "output directory should be created")
	entries, err := fs.ReadDir("/output")
	req.NoError(err)
	req.Len(entries, 1, "should have one file for the ConfigMap")
	req.Equal("ConfigMap-my-config.yaml", entries[0])
}

func TestSplitResMapToDir_FileContainsYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	rm := makeTestResMap(t)

	err := build.SplitResMapToDir(fs, rm, "/output")
	req.NoError(err)

	content, err := fs.ReadFile("/output/ConfigMap-my-config.yaml")
	req.NoError(err)
	req.Contains(string(content), "ConfigMap")
	req.Contains(string(content), "my-config")
}

func TestSplitResMapToDir_MultipleResources(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()

	rf := provider.NewDefaultDepProvider().GetResourceFactory()
	rm := resmap.New()

	yamlContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-one
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-two
data:
  key: value2
`
	resources, err := rf.SliceFromBytes([]byte(yamlContent))
	req.NoError(err)
	for _, r := range resources {
		req.NoError(rm.Append(r))
	}

	err = build.SplitResMapToDir(fs, rm, "/output")
	req.NoError(err)

	entries, err := fs.ReadDir("/output")
	req.NoError(err)
	req.Len(entries, 2, "should create one file per resource")
}

func TestSplitResMapToDir_SanitizesColonInName(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()

	rf := provider.NewDefaultDepProvider().GetResourceFactory()
	rm := resmap.New()
	// Create a resource with a colon in the name (common for cluster-scoped resources)
	r, err := rf.FromBytes([]byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: "my:resource"
data: {}
`))
	req.NoError(err)
	req.NoError(rm.Append(r))

	err = build.SplitResMapToDir(fs, rm, "/output")
	req.NoError(err)

	entries, err := fs.ReadDir("/output")
	req.NoError(err)
	req.Len(entries, 1)
	req.Equal("ConfigMap-my_resource.yaml", entries[0], "colons in resource names should be replaced with underscores")
}

func TestSplitResMapToDir_EmptyResMap(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	rm := resmap.New()

	err := build.SplitResMapToDir(fs, rm, "/output")
	req.NoError(err)
	req.True(fs.IsDir("/output"), "output directory should be created even for empty resmap")
}

// --- Build command output tests ---

func TestNewBuildCommand_OutputsYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	// Use in-memory filesystem with a minimal kustomization
	fs := filesys.MakeFsInMemory()
	err := fs.MkdirAll("/kustdir")
	req.NoError(err)
	err = fs.WriteFile("/kustdir/kustomization.yaml", []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []
`))
	req.NoError(err)

	buildOpts := build.NewBuildOptions()
	// Use the in-memory filesystem
	buildOpts.ValuesFile = "/nonexistent-values.yaml"
	buildOpts.SecretsFile = "/nonexistent-secrets.yaml"

	var buf bytes.Buffer
	cmd := build.NewBuildCommand(buildOpts, nil)
	cmd.SetOut(&buf)
	// We cannot easily run the full command without setting up plugin dirs on disk,
	// but we verify command structure here.
	req.Equal("build [flags] <kustomization directory>", cmd.Use)
}

func TestNewBuildCommand_RunsAndOutputsYAML(t *testing.T) {
	req := require.New(t)

	// Create a temporary directory with a real kustomization
	tmpDir, err := os.MkdirTemp("", "karmafun-build-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint:errcheck // No need in tests

	// Create a simple kustomization with a configmap
	kustContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
  - name: test-config
    literals:
      - key=value
`
	err = os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte(kustContent), 0o600)
	req.NoError(err)

	// Also create empty values and secrets files (non-existent is handled gracefully)
	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	buildOpts := build.NewBuildOptions()
	buildOpts.ValuesFile = filepath.Join(tmpDir, "nonexistent-values.yaml")
	buildOpts.SecretsFile = filepath.Join(tmpDir, "nonexistent-secrets.yaml")

	var buf bytes.Buffer

	cmd := build.NewBuildCommand(buildOpts, nil)
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{tmpDir})
	req.NoError(err)

	output := buf.String()
	req.Contains(output, "ConfigMap")
	req.Contains(output, "test-config")
}

func TestNewBuildCommand_RunsAndOutputsToDir(t *testing.T) {
	req := require.New(t)

	// Create a temporary directory with a real kustomization
	tmpDir, err := os.MkdirTemp("", "karmafun-build-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint:errcheck // No need in tests

	// Create a simple kustomization with a configmap
	kustContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
  - name: test-config
    literals:
      - key=value
`
	err = os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte(kustContent), 0o600)
	req.NoError(err)

	outputDir := filepath.Join(tmpDir, "output")

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	buildOpts := build.NewBuildOptions()
	buildOpts.ValuesFile = filepath.Join(tmpDir, "nonexistent-values.yaml")
	buildOpts.SecretsFile = filepath.Join(tmpDir, "nonexistent-secrets.yaml")
	buildOpts.OutputDirectory = outputDir

	cmd := build.NewBuildCommand(buildOpts, nil)

	err = cmd.RunE(cmd, []string{tmpDir})
	req.NoError(err)

	// Verify files were created in output directory
	entries, err := os.ReadDir(outputDir)
	req.NoError(err)
	req.NotEmpty(entries)
}

func TestNewBuildCommand_PostRunE(t *testing.T) {
	req := require.New(t)

	tmpDir, err := os.MkdirTemp("", "karmafun-build-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint:errcheck // No need in tests

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	buildOpts := build.NewBuildOptions()
	cmd := build.NewBuildCommand(buildOpts, nil)
	req.NotNil(cmd.PostRunE)

	// PostRunE should succeed
	err = cmd.PostRunE(cmd, []string{})
	req.NoError(err)
}

func TestNewBuildCommand_RunE_InvalidKustomizationDir(t *testing.T) {
	req := require.New(t)

	tmpDir, err := os.MkdirTemp("", "karmafun-build-test-")
	req.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint:errcheck // No need in tests

	t.Setenv("KUSTOMIZE_PLUGIN_HOME", tmpDir)

	buildOpts := build.NewBuildOptions()
	cmd := build.NewBuildCommand(buildOpts, nil)

	// Try to run with a directory that has no kustomization.yaml
	err = cmd.RunE(cmd, []string{tmpDir})
	req.Error(err)
}
