package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestResolveLinuxDataDir(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/xdg")
		got, err := ResolveLinuxDataDir()
		require.NoError(t, err)
		assert.Equal(t, "/custom/xdg", got)
	})
	t.Run("XDG_DATA_HOME empty falls back to HOME/.local/share", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "/home/testuser")
		got, err := ResolveLinuxDataDir()
		require.NoError(t, err)
		assert.Equal(t, "/home/testuser/.local/share", got)
	})
}

func TestResolveDarwinDataDir(t *testing.T) {
	t.Setenv("HOME", "/Users/testuser")
	got, err := ResolveDarwinDataDir()
	require.NoError(t, err)
	assert.Equal(t, "/Users/testuser/Library/Application Support", got)
}

func TestResolveWindowsDataDir(t *testing.T) {
	t.Run("APPDATA set", func(t *testing.T) {
		t.Setenv("APPDATA", `C:\Users\testuser\AppData\Roaming`)
		got, err := ResolveWindowsDataDir()
		require.NoError(t, err)
		assert.Equal(t, `C:\Users\testuser\AppData\Roaming`, got)
	})
	t.Run("APPDATA empty returns error", func(t *testing.T) {
		t.Setenv("APPDATA", "")
		_, err := ResolveWindowsDataDir()
		assert.ErrorIs(t, err, ErrDataDirNotResolvable)
	})
}

func TestResolveLocalPackageDirPath(t *testing.T) {
	t.Run("TYPST_PACKAGE_PATH overrides entirely", func(t *testing.T) {
		t.Setenv("TYPST_PACKAGE_PATH", "/custom/typst/packages")
		got, err := ResolveLocalPackageDirPath()
		require.NoError(t, err)
		assert.Equal(t, "/custom/typst/packages", got)
	})
	t.Run("no override ends with typst/packages", func(t *testing.T) {
		t.Setenv("TYPST_PACKAGE_PATH", "")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", "")
		got, err := ResolveLocalPackageDirPath()
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(got))
		assert.Equal(t, "packages", filepath.Base(got))
		assert.Equal(t, "typst", filepath.Base(filepath.Dir(got)))
	})
}

func TestResolveLocalPackageDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", "")
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", "")

	got, err := ResolveLocalPackageDir()
	require.NoError(t, err)

	info, statErr := os.Stat(got)
	require.NoError(t, statErr, "ResolveLocalPackageDir must create the directory")
	assert.True(t, info.IsDir())
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates nested dirs", func(t *testing.T) {
		tmp := t.TempDir()
		target := filepath.Join(tmp, "a", "b", "c")
		require.NoError(t, EnsureDir(target))
		info, err := os.Stat(target)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
	t.Run("idempotent on existing dir", func(t *testing.T) {
		tmp := t.TempDir()
		require.NoError(t, EnsureDir(tmp))
		require.NoError(t, EnsureDir(tmp), "second call must not error")
	})
}
