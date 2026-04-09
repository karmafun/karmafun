package utils_test

// cSpell: words pflag myflag mysection testcmd testapp
import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/karmafun/karmafun/pkg/utils"
)

func TestGetBaseDirectory(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	// The test must run from a directory inside a git repo (the karmafun repo itself).
	dir, err := utils.GetBaseDirectory()
	req.NoError(err)
	req.NotEmpty(dir)
	// The returned directory should contain a .git directory.
	_, err = os.Stat(dir + "/.git")
	req.NoError(err, "base directory should contain a .git directory")
}

func TestGetBaseDirectory_OutsideGitRepo(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	// Change to a temp dir that is not inside a git repo.
	tmpDir, err := os.MkdirTemp("", "no-git-")
	req.NoError(err)
	defer func() {
		req.NoError(os.RemoveAll(tmpDir))
	}()

	originalDir, err := os.Getwd()
	req.NoError(err)
	defer func() {
		req.NoError(os.Chdir(originalDir))
	}()

	req.NoError(os.Chdir(tmpDir))
	dir, err := utils.GetBaseDirectory()
	req.NoError(err)
	req.Equal(".", dir, "should return '.' when no git dir found")
}

// --- CommandConfigSection tests ---

func TestSetCommandConfigSection_SetsAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	utils.SetCommandConfigSection(cmd, "my-section")
	req.Equal("my-section", cmd.Annotations[utils.ConfigSectionAnnotation])
}

func TestCommandHasConfigSection_True(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	utils.SetCommandConfigSection(cmd, "my-section")
	req.True(utils.CommandHasConfigSection(cmd))
}

func TestCommandHasConfigSection_False_NoAnnotations(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	req.False(utils.CommandHasConfigSection(cmd))
}

func TestCommandConfigSection_ReturnsSection(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	utils.SetCommandConfigSection(cmd, "my-section")
	req.Equal("my-section", utils.CommandConfigSection(cmd))
}

func TestCommandConfigSection_ReturnsEmpty_WhenNoAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	req.Empty(utils.CommandConfigSection(cmd))
}

// --- SetSkipViperBind tests ---

func TestSetSkipViperBindForCommand_Skip(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	utils.SetSkipViperBindForCommand(cmd, true)
	req.True(utils.CmdShouldSkipViperBind(cmd))
}

func TestSetSkipViperBindForCommand_NoSkip(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	utils.SetSkipViperBindForCommand(cmd, true)
	utils.SetSkipViperBindForCommand(cmd, false)
	req.False(utils.CmdShouldSkipViperBind(cmd))
}

func TestCmdShouldSkipViperBind_False_NoAnnotations(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "test"}
	req.False(utils.CmdShouldSkipViperBind(cmd))
}

func TestSetSkipViperBindForFlag_Skip(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("myflag", "", "test flag")
	flag := flags.Lookup("myflag")
	req.NotNil(flag)
	utils.SetSkipViperBindForFlag(flag, true)
	req.True(utils.FlagShouldSkipViperBind(flag))
}

func TestSetSkipViperBindForFlag_NoSkip(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("myflag", "", "test flag")
	flag := flags.Lookup("myflag")
	req.NotNil(flag)
	utils.SetSkipViperBindForFlag(flag, true)
	utils.SetSkipViperBindForFlag(flag, false)
	req.False(utils.FlagShouldSkipViperBind(flag))
}

func TestFlagShouldSkipViperBind_False_NoAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("myflag", "", "test flag")
	flag := flags.Lookup("myflag")
	req.NotNil(flag)
	req.False(utils.FlagShouldSkipViperBind(flag))
}

// --- BindFlagValue tests ---

func TestBindFlagValue_AppliesViperValueToFlag(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_flag", "from-viper")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("my-flag", "default", "test flag")
	flag := flags.Lookup("my-flag")
	req.NotNil(flag)

	err := utils.BindFlagValue(flag, v, "my_flag")
	req.NoError(err)
	req.Equal("from-viper", flag.Value.String())
}

func TestBindFlagValue_SkipsWhenFlagChanged(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_flag", "from-viper")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("my-flag", "default", "test flag")
	err := flags.Set("my-flag", "from-cli")
	req.NoError(err)
	flag := flags.Lookup("my-flag")
	req.NotNil(flag)
	req.True(flag.Changed)

	err = utils.BindFlagValue(flag, v, "my_flag")
	req.NoError(err)
	req.Equal("from-cli", flag.Value.String(), "flag changed by CLI should not be overridden by viper")
}

func TestBindFlagValue_SkipsWhenViperKeyNotSet(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("my-flag", "default", "test flag")
	flag := flags.Lookup("my-flag")
	req.NotNil(flag)

	err := utils.BindFlagValue(flag, v, "my_flag")
	req.NoError(err)
	req.Equal("default", flag.Value.String(), "should keep default when viper key is not set")
}

func TestBindFlagValue_SliceFlag(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_slice", []string{"a", "b", "c"})

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringSlice("my-slice", []string{}, "test slice flag")
	flag := flags.Lookup("my-slice")
	req.NotNil(flag)

	err := utils.BindFlagValue(flag, v, "my_slice")
	req.NoError(err)
	req.Equal("[a,b,c]", flag.Value.String())
}

// --- AddConfigFlag tests ---

func TestAddConfigFlag_AddsFlag(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := &cobra.Command{Use: "testcmd"}
	utils.AddConfigFlag(cmd)
	flag := cmd.PersistentFlags().Lookup(utils.ConfigFlag)
	req.NotNil(flag, "config flag should be added to command")
	req.Equal("c", flag.Shorthand)
}

// --- BindFlagsToViper / ApplyViperConfigToFlags ---

func TestBindFlags_SkipsCommandWithSkipAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_flag", "from-viper")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("my-flag", "default", "test flag")
	utils.SetSkipViperBindForCommand(cmd, true)

	utils.ApplyViperConfigToFlags(cmd, v)

	flag := cmd.Flags().Lookup("my-flag")
	req.NotNil(flag)
	req.Equal("default", flag.Value.String(), "flag in skipped command should not be changed")
}

func TestBindFlags_AppliesViperValuesToFlags(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_flag", "from-viper")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("my-flag", "default", "test flag")

	utils.ApplyViperConfigToFlags(cmd, v)

	flag := cmd.Flags().Lookup("my-flag")
	req.NotNil(flag)
	req.Equal("from-viper", flag.Value.String())
}

func TestBindFlags_SkipsFlagWithSkipAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("my_flag", "from-viper")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("my-flag", "default", "test flag")
	flag := cmd.Flags().Lookup("my-flag")
	req.NotNil(flag)
	utils.SetSkipViperBindForFlag(flag, true)

	utils.ApplyViperConfigToFlags(cmd, v)

	req.Equal("default", flag.Value.String(), "flag with skip annotation should not be changed")
}

func TestBindFlags_UsesConfigSectionAsPrefix(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("mysection.my_flag", "from-section")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("my-flag", "default", "test flag")
	utils.SetCommandConfigSection(cmd, "mysection")

	utils.ApplyViperConfigToFlags(cmd, v)

	flag := cmd.Flags().Lookup("my-flag")
	req.NotNil(flag)
	req.Equal("from-section", flag.Value.String())
}

func TestBindFlags_Recurses_Subcommands(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()
	v.Set("child_flag", "child-from-viper")

	parent := &cobra.Command{Use: "parent"}
	child := &cobra.Command{Use: "child"}
	child.Flags().String("child-flag", "default", "child test flag")
	parent.AddCommand(child)

	utils.ApplyViperConfigToFlags(parent, v)

	flag := child.Flags().Lookup("child-flag")
	req.NotNil(flag)
	req.Equal("child-from-viper", flag.Value.String())
}

// --- BindFlag tests ---

func TestBindFlag_BindsPersistentFlag(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("my-flag", "default", "test flag")
	flag := flags.Lookup("my-flag")
	req.NotNil(flag)

	err := utils.BindFlag(flag, v, "my_flag")
	req.NoError(err)

	v.Set("my_flag", "from-viper")
	req.Equal("from-viper", v.GetString("my_flag"))
}

// --- BindFlagsToViper tests ---

func TestBindFlagsToViper_BindsFlags(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	v := viper.New()

	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("my-flag", "default", "test flag")

	utils.BindFlagsToViper(cmd, v)
	// After binding, setting viper value should be accessible
	v.Set("my_flag", "bound-value")
	req.Equal("bound-value", v.GetString("my_flag"))
}

// --- InitializeConfiguration tests ---

func TestInitializeConfiguration_NoConfigFile(t *testing.T) {
	req := require.New(t)
	v := viper.New()

	rootCmd := &cobra.Command{Use: "testapp"}
	utils.AddConfigFlag(rootCmd)

	err := utils.InitializeConfiguration(rootCmd, v)
	req.NoError(err)
}

func TestInitializeConfiguration_WithConfigFileFlag(t *testing.T) {
	req := require.New(t)
	v := viper.New()

	rootCmd := &cobra.Command{Use: "testapp"}
	utils.AddConfigFlag(rootCmd)

	// Set config flag to a non-existent file - should not error since ReadInConfig is best-effort
	err := rootCmd.PersistentFlags().Set(utils.ConfigFlag, "/tmp/nonexistent-config-file-karmafun.yaml")
	req.NoError(err)

	err = utils.InitializeConfiguration(rootCmd, v)
	req.NoError(err)
}

func TestInitializeConfiguration_WithEnvVars(t *testing.T) {
	t.Setenv("TESTAPP_MY_FLAG", "env-value")
	req := require.New(t)
	v := viper.New()

	rootCmd := &cobra.Command{Use: "testapp"}
	utils.AddConfigFlag(rootCmd)
	rootCmd.Flags().String("my-flag", "default", "test flag")

	err := utils.InitializeConfiguration(rootCmd, v)
	req.NoError(err)
}
