package cmd //nolint: testpackage

import (
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestInstallDirIsRejectedByTheProjectCommands pins where an install dir may be
// named. It receives one package's files directly, so the commands that install
// or delete a whole dependency graph must not accept it.
func TestInstallDirIsRejectedByTheProjectCommands(t *testing.T) {
	t.Parallel()
	for _, cmd := range []*cobra.Command{addCmd, syncCmd, removeCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			t.Parallel()
			err := cmd.ParseFlags([]string{"--" + paths.InstallDirFlag, "./dist"})
			require.ErrorContains(t, err, "unknown flag")
		})
	}
}

// TestInstallDirIsAcceptedByTheSinglePackageCommands is the other half: the
// commands that operate on one package keep the flag.
func TestInstallDirIsAcceptedByTheSinglePackageCommands(t *testing.T) {
	t.Parallel()
	for _, cmd := range []*cobra.Command{installCmd, uninstallCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, cmd.ParseFlags([]string{"--" + paths.InstallDirFlag, "./dist"}))
		})
	}
}

func TestFirstArg(t *testing.T) {
	t.Parallel()
	require.Empty(t, firstArg(nil))
	require.Equal(t, "a", firstArg([]string{"a", "b"}))
}

func TestMustPassesTheValueThrough(t *testing.T) {
	t.Parallel()
	require.Equal(t, 42, Must(42, nil))
}

func TestNewLoggerToleratesACommandWithoutVerboseFlag(t *testing.T) {
	t.Parallel()
	require.NotNil(t, newLogger(&cobra.Command{}))

	withFlag := &cobra.Command{}
	withFlag.Flags().CountP("verbose", "v", "")
	require.NoError(t, withFlag.ParseFlags([]string{"-vv"}))
	require.NotNil(t, newLogger(withFlag))
}

func TestExecuteRunsTheRootCommand(t *testing.T) { //nolint: paralleltest // rootCmd is shared
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	Execute()
}
