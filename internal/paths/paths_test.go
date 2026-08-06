package paths_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallDir(t *testing.T) {
	t.Run("flag overrides path", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "")
		path, overridden, err := paths.InstallDir("/flag/path")
		require.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/flag/path", path)
	})
	t.Run("env var overrides path", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "/env/path")
		path, overridden, err := paths.InstallDir("")
		require.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/env/path", path)
	})
	t.Run("flag takes precedence over env var", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "/env/path")
		path, overridden, err := paths.InstallDir("/flag/path")
		require.NoError(t, err)
		assert.True(t, overridden)
		assert.Equal(t, "/flag/path", path)
	})
	t.Run("no override falls back to OS default", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "")
		_, overridden, err := paths.InstallDir("")
		require.NoError(t, err)
		assert.False(t, overridden)
	})
}

func TestLinuxDataDir(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/xdg")
		got, err := paths.LinuxDataDir()
		require.NoError(t, err)
		assert.Equal(t, "/custom/xdg", got)
	})
	t.Run("XDG_DATA_HOME empty falls back to HOME/.local/share", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "/home/testuser")
		got, err := paths.LinuxDataDir()
		require.NoError(t, err)
		assert.Equal(t, "/home/testuser/.local/share", got)
	})
}

func TestDarwinDataDir(t *testing.T) {
	t.Setenv("HOME", "/Users/testuser")
	got, err := paths.DarwinDataDir()
	require.NoError(t, err)
	assert.Equal(t, "/Users/testuser/Library/Application Support", got)
}

func TestWindowsDataDir(t *testing.T) {
	t.Run("APPDATA set", func(t *testing.T) {
		t.Setenv("APPDATA", `C:\Users\testuser\AppData\Roaming`)
		got, err := paths.WindowsDataDir()
		require.NoError(t, err)
		assert.Equal(t, `C:\Users\testuser\AppData\Roaming`, got)
	})
	t.Run("APPDATA empty returns error", func(t *testing.T) {
		t.Setenv("APPDATA", "")
		_, err := paths.WindowsDataDir()
		assert.ErrorIs(t, err, paths.ErrDataDirNotResolvable)
	})
}

func TestTypstPackagesDir(t *testing.T) {
	t.Run("TYPST_PACKAGE_PATH overrides entirely", func(t *testing.T) {
		t.Setenv("TYPST_PACKAGE_PATH", "/custom/typst/packages")
		got, err := paths.TypstPackagesDir()
		require.NoError(t, err)
		assert.Equal(t, "/custom/typst/packages", got)
	})
	t.Run("no override ends with typst/packages", func(t *testing.T) {
		t.Setenv("TYPST_PACKAGE_PATH", "")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("APPDATA", t.TempDir())
		got, err := paths.TypstPackagesDir()
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(got))
		assert.Equal(t, "packages", filepath.Base(got))
		assert.Equal(t, "typst", filepath.Base(filepath.Dir(got)))
	})
}

func TestEnsureTypstPackagesDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", "")
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", tmp)

	got, err := paths.EnsureTypstPackagesDir()
	require.NoError(t, err)

	info, statErr := os.Stat(got)
	require.NoError(t, statErr, "EnsureTypstPackagesDir must create the directory")
	assert.True(t, info.IsDir())
}

func TestEnsureDir(t *testing.T) {
	t.Parallel()
	t.Run("creates nested dirs", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		target := filepath.Join(tmp, "a", "b", "c")
		require.NoError(t, paths.EnsureDir(target))
		info, err := os.Stat(target)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
	t.Run("idempotent on existing dir", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		require.NoError(t, paths.EnsureDir(tmp))
		require.NoError(t, paths.EnsureDir(tmp), "second call must not error")
	})
}

func TestIsDir(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	file := filepath.Join(tmp, "a.typ")
	require.NoError(t, paths.WriteFile(file, []byte("x")))

	assert.True(t, paths.IsDir(tmp))
	assert.False(t, paths.IsDir(file))
	assert.False(t, paths.IsDir(filepath.Join(tmp, "missing")))
}

func TestWriteFile(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "out.typ")
	require.NoError(t, paths.WriteFile(file, []byte("content")))

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}
