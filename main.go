package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"

	"github.com/karmafun/karmafun/pkg/cmd/build"
	"github.com/karmafun/karmafun/pkg/cmd/setup"
	"github.com/karmafun/karmafun/pkg/process"
	"github.com/karmafun/karmafun/pkg/utils"
)

var (
	KarmafunVersion = "v0.4.3" // <---VERSION--->
	Commit          = "unknown"
	BuildDate       = "unknown"
	BuiltBy         = "unknown"
)

func main() { // nocov
	logOptions := build.NewLogOptions()
	cmd := command.Build(process.NewDefaultProcessor(), command.StandaloneEnabled, false)
	cmd.Use = "karmafun"
	cmd.Version = KarmafunVersion
	command.AddGenerateDockerfile(cmd)
	cmd.AddCommand(build.NewBuildCommand(nil, logOptions))
	cmd.AddCommand(setup.NewSetupCommand(nil))
	logOptions.AddFlags(cmd.PersistentFlags())
	utils.AddConfigFlag(cmd)
	var calledAs string
	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		calledAs = cmd.CalledAs()
		rootCmd := cmd.Root()
		err := utils.InitializeConfiguration(rootCmd, viper.GetViper())
		if err != nil {
			return fmt.Errorf("initializing configuration: %w", err)
		}
		logOptions.InitLogger()
		return nil
	}

	utils.BindFlagsToViper(cmd, viper.GetViper())

	if err := cmd.Execute(); err != nil {
		if calledAs == "build" {
			cobra.CheckErr(err)
		}
		os.Exit(1)
	}
}
