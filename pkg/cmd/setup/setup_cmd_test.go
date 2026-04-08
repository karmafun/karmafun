package setup_test

// cSpell: words filesys testify karmafun

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/cmd/setup"
)

func TestNewSetupCommand_CreatesCommand(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := setup.NewSetupCommand(nil)
	req.NotNil(cmd)
	req.Equal("setup", cmd.Use)
	req.NotEmpty(cmd.Short)
	req.NotEmpty(cmd.Long)
}

func TestNewSetupCommand_HasInstallAlias(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := setup.NewSetupCommand(nil)
	req.Contains(cmd.Aliases, "install")
}

func TestNewSetupCommand_AcceptsNoArgs(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := setup.NewSetupCommand(nil)
	err := cmd.Args(cmd, []string{})
	req.NoError(err)
}

func TestNewSetupCommand_RejectsArgs(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cmd := setup.NewSetupCommand(nil)
	err := cmd.Args(cmd, []string{"extra-arg"})
	req.Error(err)
}

func TestNewSetupCommand_NilFs_UsesOnDisk(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	// Should not panic when fs is nil (uses on-disk filesystem by default)
	cmd := setup.NewSetupCommand(nil)
	req.NotNil(cmd)
}

func TestNewSetupCommand_WithInMemoryFs(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	cmd := setup.NewSetupCommand(fs)
	req.NotNil(cmd)
}
