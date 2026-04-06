package build

// cSpell: words filesys pflag wrapcheck gosec
import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
	"github.com/karmafun/karmafun/pkg/templates"
)

const (
	defaultValuesFile  = "values.yaml"
	defaultSecretsFile = "secrets.sops.yaml" //nolint:gosec // This file is expected to contain encrypted secrets.
)

type LogOptions struct {
	Level string
	Json  bool
}

func NewLogOptions() *LogOptions {
	return &LogOptions{
		Level: "info",
	}
}

func (l *LogOptions) InitLogger() {
	// set log level
	var level slog.Level
	err := level.UnmarshalText([]byte(l.Level))
	if err != nil {
		level = slog.LevelInfo
	}

	var handler slog.Handler
	if l.Json {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}

type BuildOptions struct {
	fs              filesys.FileSystem
	out             io.Writer
	ValuesFile      string
	SecretsFile     string
	OutputDirectory string
}

func NewBuildOptions() *BuildOptions {
	return &BuildOptions{
		fs:          filesys.MakeFsOnDisk(),
		ValuesFile:  defaultValuesFile,
		SecretsFile: defaultSecretsFile,
	}
}

func (o *BuildOptions) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.ValuesFile, "values-file", o.ValuesFile, "the file containing the values for the function")
	flags.StringVar(&o.SecretsFile, "secrets-file", o.SecretsFile, "the file containing the secrets for the function")
	flags.StringVarP(
		&o.OutputDirectory,
		"output-directory",
		"o",
		o.OutputDirectory,
		"the directory to write the output resources to",
	)
}

func (l *LogOptions) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&l.Level, "log-level", "info", "the log level to use (debug, info, warn, error)")
	flags.BoolVar(&l.Json, "log-json", false, "whether to log in JSON format")
}

// NewBuildCommand returns a cobra.Command to run the build command.
func NewBuildCommand(buildOptions *BuildOptions, logOptions *LogOptions) *cobra.Command {
	if buildOptions == nil {
		buildOptions = NewBuildOptions()
	}
	createdLogOptions := false
	if logOptions == nil {
		logOptions = NewLogOptions()
		createdLogOptions = true
	}
	cmd := &cobra.Command{
		Use:   "build [flags] <kustomization directory>",
		Args:  cobra.ExactArgs(1),
		Short: "Build a kustomization with values and secrets injection allowing use of karmafun plugins.",
		Long: "Build a kustomization with values and secrets injection allowing use of karmafun plugins.\n\n" +
			"The build command runs the kustomization in the specified directory with the provided values and " +
			"secrets files.\n" +
			"Any plugin that can be used in a kustomization can be used in the specified directory, allowing you to " +
			"use karmafun plugins as well as any kustomize plugin.\n" +
			"The output resources are written to the specified output directory. " +
			"If no output directory is specified, the output resources are printed to STDOUT.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if createdLogOptions {
				logOptions.InitLogger()
			}
			err := plugins.CreatePluginDirectoryHierarchy(buildOptions.fs)
			if err != nil {
				return fmt.Errorf("while creating plugin directory hierarchy: %w", err)
			}
			buildOptions.out = cmd.OutOrStdout()
			return runBuildCommand(buildOptions, args[0])
		},
		PostRunE: func(_ *cobra.Command, _ []string) error {
			slog.Debug("Removing plugin dir...")
			err := plugins.RemovePluginDirectoryHierarchy(buildOptions.fs)
			if err != nil {
				return fmt.Errorf("while removing plugin directory hierarchy: %w", err)
			}
			return nil
		},
	}
	buildOptions.AddFlags(cmd.Flags())
	if createdLogOptions {
		logOptions.AddFlags(cmd.Flags())
	}
	return cmd
}

func SplitResMapToDir(fs filesys.FileSystem, resources resmap.ResMap, destDir string) error {
	if err := fs.MkdirAll(destDir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}
	for _, resource := range resources.Resources() {
		kind := resource.GetKind()
		name := resource.GetName()

		yamlData, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource %s/%s: %w", kind, name, err)
		}

		safeName := strings.ReplaceAll(name, ":", "_")
		filename := fmt.Sprintf("%s-%s.yaml", kind, safeName)
		path := filepath.Join(destDir, filename)

		if err := fs.WriteFile(path, yamlData); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
		fmt.Fprintf(os.Stderr, "Created: %s\n", path)
	}
	return nil
}

func runBuildCommand(buildOptions *BuildOptions, directory string) error {
	values, err := templates.ReadPlatformValues(buildOptions.fs, buildOptions.ValuesFile)
	if err != nil {
		return fmt.Errorf("while reading values from file %s: %w", buildOptions.ValuesFile, err)
	}
	secrets, err := templates.ReadSecretsValues(buildOptions.fs, buildOptions.SecretsFile)
	if err != nil {
		return fmt.Errorf("while reading secrets from file %s: %w", buildOptions.SecretsFile, err)
	}

	values = templates.MergeValues(values, secrets)

	templateFs := templates.NewTemplateFS(buildOptions.fs, values.AsMap())

	resMap, err := extras.RunKustomizations(templateFs, directory)
	if err != nil {
		return err //nolint:wrapcheck // No added value
	}
	if buildOptions.OutputDirectory != "" {
		err = SplitResMapToDir(buildOptions.fs, resMap, buildOptions.OutputDirectory)
		if err != nil {
			return fmt.Errorf("while splitting resources to directory %s: %w", buildOptions.OutputDirectory, err)
		}
	} else {
		content, err := resMap.AsYaml()
		if err != nil {
			return fmt.Errorf("while converting resources to YAML in %s: %w", directory, err)
		}
		_, err = buildOptions.out.Write(content)
		if err != nil {
			return fmt.Errorf("while writing output: %w", err)
		}
	}
	// Do something with content, e.g., print or save the resources
	return nil
}
