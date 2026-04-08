package setup

// cSpell: words filesys

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/plugins"
)

// NewSetupCommand returns a cobra.Command to run the setup (install) command.
func NewSetupCommand(fs filesys.FileSystem) *cobra.Command {
	if fs == nil {
		fs = filesys.MakeFsOnDisk()
	}
	cmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"install"},
		Short:   "Create symlinks in the kustomize plugin directory for all karmafun plugin kinds.",
		Long: "Create symlinks in the kustomize plugin directory for all karmafun plugin kinds.\n\n" +
			"This command creates the required symlinks under the kustomize plugin directory so that\n" +
			"kustomize can discover all plugin kinds provided by karmafun.\n\n" +
			"This is only needed when using karmafun as a legacy exec plugin (i.e., without\n" +
			"the config.kubernetes.io/function annotation).",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := plugins.CreatePluginDirectoryHierarchy(fs); err != nil {
				return fmt.Errorf("while creating plugin directory hierarchy: %w", err)
			}
			return nil
		},
	}
	return cmd
}
