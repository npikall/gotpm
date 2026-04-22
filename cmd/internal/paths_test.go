package internal

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newCmdWithInstallDir(t *testing.T, value string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String(InstallDirFlag, "", "")
	if value != "" {
		if err := cmd.Flags().Set(InstallDirFlag, value); err != nil {
			t.Fatalf("setting flag %q: %v", InstallDirFlag, err)
		}
	}
	return cmd
}

func Test_ResolvePackageDirPath(t *testing.T) {
	t.Run("flag overrides path", func(t *testing.T) {
		t.Setenv(InstallDirEnvVar, "")
		cmd := newCmdWithInstallDir(t, "/flag/path")
		path, overridden, err := ResolvePackageDirPath(cmd)
		assert.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/flag/path", path)
	})
	t.Run("env var overrides path", func(t *testing.T) {
		t.Setenv(InstallDirEnvVar, "/env/path")
		cmd := newCmdWithInstallDir(t, "")
		path, overridden, err := ResolvePackageDirPath(cmd)
		assert.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/env/path", path)
	})
	t.Run("flag takes precedence over env var", func(t *testing.T) {
		t.Setenv(InstallDirEnvVar, "/env/path")
		cmd := newCmdWithInstallDir(t, "/flag/path")
		path, overridden, err := ResolvePackageDirPath(cmd)
		assert.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/flag/path", path)
	})
	t.Run("no override falls back to OS default", func(t *testing.T) {
		t.Setenv(InstallDirEnvVar, "")
		cmd := newCmdWithInstallDir(t, "")
		_, overridden, err := ResolvePackageDirPath(cmd)
		assert.NoError(t, err)
		assert.False(t, overridden)
	})
}
