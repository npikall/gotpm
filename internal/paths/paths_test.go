package paths_test

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestDataDirFor(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/xdg")
	t.Setenv("HOME", "/Users/testuser")
	t.Setenv("APPDATA", `C:\Users\testuser\AppData\Roaming`)

	for goos, want := range map[string]string{
		"linux":   "/custom/xdg",
		"darwin":  "/Users/testuser/Library/Application Support",
		"windows": `C:\Users\testuser\AppData\Roaming`,
	} {
		got, err := paths.DataDirFor(goos)
		require.NoError(t, err, goos)
		assert.Equal(t, want, got, goos)
	}

	_, err := paths.DataDirFor("plan9")
	require.ErrorIs(t, err, paths.ErrDataDirNotResolvable)
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

func TestRemove(t *testing.T) {
	t.Parallel()
	t.Run("removes regular directory and its contents", func(t *testing.T) {
		t.Parallel()
		parent := t.TempDir()
		target := filepath.Join(parent, "pkg")
		require.NoError(t, paths.EnsureDir(filepath.Join(target, "sub")))
		require.NoError(t, paths.WriteFile(filepath.Join(target, "lib.typ"), []byte("")))

		require.NoError(t, paths.Remove(target))
		assert.NoDirExists(t, target)
	})
	t.Run("removes symlink without deleting the pointed-to directory", func(t *testing.T) {
		t.Parallel()
		actual := t.TempDir()
		require.NoError(t, paths.WriteFile(filepath.Join(actual, "lib.typ"), []byte("")))
		link := filepath.Join(t.TempDir(), "link")
		require.NoError(t, os.Symlink(actual, link))

		require.NoError(t, paths.Remove(link))
		assert.NoFileExists(t, link)
		assert.DirExists(t, actual, "the link target must be untouched")
	})
	t.Run("removes dangling symlink", func(t *testing.T) {
		t.Parallel()
		parent := t.TempDir()
		link := filepath.Join(parent, "dangling")
		require.NoError(t, os.Symlink(filepath.Join(parent, "nowhere"), link))

		require.NoError(t, paths.Remove(link))
		assert.NoFileExists(t, link)
	})
	t.Run("missing path reports an error", func(t *testing.T) {
		t.Parallel()
		err := paths.Remove(filepath.Join(t.TempDir(), "nope"))
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestSize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, paths.EnsureDir(filepath.Join(dir, "sub")))
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "a.typ"), []byte("abc")))
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "sub", "b.typ"), []byte("defg")))
	file := filepath.Join(t.TempDir(), "c.typ")
	require.NoError(t, paths.WriteFile(file, []byte("hi")))

	size, err := paths.Size(dir, file, filepath.Join(dir, "missing"))

	require.NoError(t, err)
	assert.Equal(t, int64(9), size, "files in nested directories and lone files count; a missing path adds nothing")
}

func TestSizeUnreadableDirectory(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions are not enforced here")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	require.NoError(t, paths.EnsureDir(locked))
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, paths.DirPerm) })

	_, err := paths.Size(dir)

	require.ErrorIs(t, err, os.ErrPermission)
}

func TestOriginEnvVar(t *testing.T) {
	t.Parallel()
	assert.Equal(t, paths.InstallDirEnvVar, paths.OriginInstallEnv.EnvVar())
	assert.Equal(t, paths.TypstPackagePathEnvVar, paths.OriginTypstEnv.EnvVar())
	assert.Empty(t, paths.OriginDefault.EnvVar(), "the default location has no variable behind it")
}

func TestFileExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, paths.WriteFile(file, []byte("x")))

	require.NoError(t, paths.FileExists(file))
	require.ErrorIs(t, paths.FileExists(dir), paths.ErrNotAFile)
	require.ErrorIs(t, paths.FileExists(filepath.Join(dir, "missing")), os.ErrNotExist)
}

func TestDirectoryExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, paths.WriteFile(file, []byte("x")))

	require.NoError(t, paths.DirectoryExists(dir))
	require.ErrorIs(t, paths.DirectoryExists(file), paths.ErrNotADirectory)
	require.ErrorIs(t, paths.DirectoryExists(filepath.Join(dir, "missing")), os.ErrNotExist)
}

func TestPackagesDir(t *testing.T) {
	t.Run("GOTPM_INSTALL_DIR wins", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "/gotpm/packages")
		t.Setenv(paths.TypstPackagePathEnvVar, "/typst/packages")
		dir, origin, err := paths.PackagesDir()
		require.NoError(t, err)
		assert.Equal(t, "/gotpm/packages", dir)
		assert.Equal(t, paths.OriginInstallEnv, origin)
	})
	t.Run("TYPST_PACKAGE_PATH comes next", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "")
		t.Setenv(paths.TypstPackagePathEnvVar, "/typst/packages")
		dir, origin, err := paths.PackagesDir()
		require.NoError(t, err)
		assert.Equal(t, "/typst/packages", dir)
		assert.Equal(t, paths.OriginTypstEnv, origin)
	})
	t.Run("otherwise the default", func(t *testing.T) {
		t.Setenv(paths.InstallDirEnvVar, "")
		t.Setenv(paths.TypstPackagePathEnvVar, "")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("APPDATA", t.TempDir())
		dir, origin, err := paths.PackagesDir()
		require.NoError(t, err)
		assert.Equal(t, "packages", filepath.Base(dir))
		assert.Equal(t, paths.OriginDefault, origin)
	})
}

func TestAppDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("APPDATA", filepath.Join(home, "appdata"))

	data, err := paths.AppDataDir("gotpm")
	require.NoError(t, err)
	assert.Equal(t, "gotpm", filepath.Base(data))

	config, err := paths.AppConfigDir("gotpm")
	require.NoError(t, err)
	assert.Equal(t, "gotpm", filepath.Base(config))
}

func TestConfigDirFailsWithoutAHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", "")

	_, err := paths.ConfigDir()
	require.Error(t, err)

	_, err = paths.AppConfigDir("gotpm")
	require.Error(t, err)
}

func TestGotpmForksDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("APPDATA", filepath.Join(home, "appdata"))

	data, err := paths.GotpmDataDir()
	require.NoError(t, err)
	forks, err := paths.GotpmForksDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(data, "forks"), forks)
}

func TestGotpmForksDirFailsWithoutADataDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", "")

	_, err := paths.GotpmForksDir()
	require.Error(t, err)
}

func TestGotpmFontsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("APPDATA", filepath.Join(home, "appdata"))

	data, err := paths.GotpmDataDir()
	require.NoError(t, err)
	fonts, err := paths.GotpmFontsDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(data, "fonts"), fonts)
}

func TestGotpmFontsDirFailsWithoutADataDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", "")

	_, err := paths.GotpmFontsDir()
	require.Error(t, err)
}
