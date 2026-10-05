package cache_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	cachecmd "github.com/npikall/gotpm/internal/cmds/cache"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateCacheDir points HOME and, critically, forces XDG_CONFIG_HOME and
// XDG_DATA_HOME to the same tree so config.toml and the cache/remotes dir
// collide the way they do by default on darwin and windows (both resolve to
// the OS "app support" dir there). This reproduces the bug scenario
// regardless of the OS actually running the test.
func isolateCacheDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	shared := filepath.Join(tmp, "shared")
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", shared)
	t.Setenv("XDG_DATA_HOME", shared)
	t.Setenv("APPDATA", shared)
}

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

func seedCacheState(t *testing.T) (remotesDir, cachePath, configPath string) { //nolint: nonamedreturns
	t.Helper()

	remotesDir, err := remote.CacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(remotesDir, "example.com", "repo"), paths.DirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(remotesDir, "example.com", "repo", "file.txt"), []byte("data"), paths.FilePerm))

	require.NoError(t, index.SaveCache(index.Index{"pkg": "1.0.0"}))
	cachePath, err = index.CachePath()
	require.NoError(t, err)

	cfg := &config.Config{}
	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))
	require.NoError(t, config.Save(cfg))
	configPath, err = config.Path()
	require.NoError(t, err)

	return remotesDir, cachePath, configPath
}

func TestClear_PreservesConfig(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	remotesDir, cachePath, configPath := seedCacheState(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{}, discardLogger()))

	assert.NoDirExists(t, remotesDir, "remotes dir must be removed")
	assert.NoFileExists(t, cachePath, "index cache must be removed")
	assert.FileExists(t, configPath, "config.toml must survive cache clear")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", cfg.Fork.Path, "config content must be intact")
}

func TestClear_MissingFilesNoOp(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)

	err := cachecmd.Clear(&cachecmd.Options{}, discardLogger())
	require.NoError(t, err, "clearing an already-empty cache must not error")
}

func TestClear_DryRunDeletesNothing(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	remotesDir, cachePath, configPath := seedCacheState(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{DryRun: true}, discardLogger()))

	assert.DirExists(t, remotesDir, "dry-run must not remove remotes dir")
	assert.FileExists(t, cachePath, "dry-run must not remove index cache")
	assert.FileExists(t, configPath, "dry-run must not touch config.toml")
}

func TestClear_FailsWithoutADataDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", "")

	require.Error(t, cachecmd.Clear(&cachecmd.Options{}, discardLogger()))
}

func TestClear_FailsOnAnUnreadableCache(t *testing.T) { //nolint: paralleltest
	if os.Geteuid() == 0 {
		t.Skip("root reads any directory")
	}
	isolateCacheDir(t)
	remotesDir, _, _ := seedCacheState(t)
	locked := filepath.Join(remotesDir, "locked")
	require.NoError(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) }) // so TempDir can clean up

	require.Error(t, cachecmd.Clear(&cachecmd.Options{}, discardLogger()))
}

func seedForkClone(t *testing.T) string {
	t.Helper()
	forksDir, err := paths.GotpmForksDir()
	require.NoError(t, err)
	clone := filepath.Join(forksDir, "github.com", "me", "packages")
	require.NoError(t, os.MkdirAll(clone, paths.DirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(clone, "file.txt"), []byte("data"), paths.FilePerm))
	return forksDir
}

func TestClear_LeavesForkClonesAlone(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	seedCacheState(t)
	forksDir := seedForkClone(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{}, discardLogger()))

	assert.DirExists(t, forksDir, "a plain cache clear must not remove fork clones")
}

func TestClear_ForksRemovesOnlyForkClones(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	remotesDir, cachePath, configPath := seedCacheState(t)
	forksDir := seedForkClone(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{Forks: true}, discardLogger()))

	assert.NoDirExists(t, forksDir, "fork clones must be removed")
	assert.DirExists(t, remotesDir, "--forks must not remove the remotes cache")
	assert.FileExists(t, cachePath, "--forks must not remove the index cache")
	assert.FileExists(t, configPath, "--forks must not touch config.toml")
}

func TestClear_ForksDryRunDeletesNothing(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	forksDir := seedForkClone(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{Forks: true, DryRun: true}, discardLogger()))

	assert.DirExists(t, forksDir, "dry-run must not remove fork clones")
}

func TestClear_ForksMissingNoOp(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{Forks: true}, discardLogger()))
}

func TestClear_ForksFailsWithoutADataDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", "")

	require.Error(t, cachecmd.Clear(&cachecmd.Options{Forks: true}, discardLogger()))
}

func TestClear_ForksFailsOnAnUnreadableClone(t *testing.T) { //nolint: paralleltest
	if os.Geteuid() == 0 {
		t.Skip("root reads any directory")
	}
	isolateCacheDir(t)
	forksDir := seedForkClone(t)
	locked := filepath.Join(forksDir, "locked")
	require.NoError(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) }) // so TempDir can clean up

	require.Error(t, cachecmd.Clear(&cachecmd.Options{Forks: true}, discardLogger()))
}

func seedFonts(t *testing.T) (string, string) {
	t.Helper()
	fontsDir, err := paths.GotpmFontsDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(fontsDir, "lato"), paths.DirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(fontsDir, "lato", "Lato.ttf"), []byte("font"), paths.FilePerm))
	fontIndex, err := fonts.IndexPath()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fontIndex, []byte("{}"), paths.FilePerm))
	return fontsDir, fontIndex
}

func TestClear_LeavesFontsAloneButDropsTheFontIndex(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	seedCacheState(t)
	fontsDir, fontIndex := seedFonts(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{}, discardLogger()))

	assert.DirExists(t, fontsDir, "the font directory is never cache")
	assert.NoFileExists(t, fontIndex, "the searched family list is cache")
}

func TestClear_FontsRemovesOnlyTheFontDirectory(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	remotesDir, cachePath, _ := seedCacheState(t)
	forksDir := seedForkClone(t)
	fontsDir, _ := seedFonts(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{Fonts: true}, discardLogger()))

	assert.NoDirExists(t, fontsDir)
	assert.DirExists(t, remotesDir)
	assert.FileExists(t, cachePath)
	assert.DirExists(t, forksDir)
}

func TestClear_FontsDryRunDeletesNothing(t *testing.T) { //nolint: paralleltest
	isolateCacheDir(t)
	fontsDir, _ := seedFonts(t)

	require.NoError(t, cachecmd.Clear(&cachecmd.Options{Fonts: true, DryRun: true}, discardLogger()))

	assert.DirExists(t, fontsDir)
}
