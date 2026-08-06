package cache_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	cachecmd "github.com/npikall/gotpm/internal/cmds/cache"
	"github.com/npikall/gotpm/internal/config"
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
