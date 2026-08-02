package config_test

import (
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGetUnset(t *testing.T) { //nolint: paralleltest
	cfg := &Config{}

	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))
	require.NoError(t, cfg.Set("fork.url", "https://github.com/user/packages"))

	path, err := cfg.Get("fork.path")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", path)

	url, err := cfg.Get("fork.url")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/packages", url)

	require.NoError(t, cfg.Unset("fork.path"))
	path, err = cfg.Get("fork.path")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestSet_UnknownKey(t *testing.T) { //nolint: paralleltest
	cfg := &Config{}
	err := cfg.Set("fork.nope", "value")
	assert.ErrorIs(t, err, ErrUnknownKey)
}

func TestSet_NotSettablePath(t *testing.T) { //nolint: paralleltest
	cfg := &Config{}
	err := cfg.Set("fork.path.extra", "value")
	assert.ErrorIs(t, err, ErrNotSettablePath)
}

func TestGet_UnknownTopLevelKey(t *testing.T) { //nolint: paralleltest
	cfg := &Config{}
	_, err := cfg.Get("nope")
	assert.ErrorIs(t, err, ErrUnknownKey)
}

func TestEntries(t *testing.T) { //nolint: paralleltest
	cfg := &Config{}
	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))

	entries := cfg.Entries()
	assert.Equal(t, []KV{
		{Key: "fork.path", Value: "/tmp/my-fork"},
		{Key: "fork.url", Value: ""},
	}, entries)
}

func TestSaveAndLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg := &Config{}
	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))
	require.NoError(t, cfg.Set("fork.url", "https://github.com/user/packages"))
	require.NoError(t, Save(cfg))

	path, err := Path()
	require.NoError(t, err)
	assert.FileExists(t, path)

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}

func TestLoad_NoFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
}

func TestPath_ContainsGoTPM(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := Path()
	require.NoError(t, err)
	assert.Equal(t, "gotpm", filepath.Base(filepath.Dir(path)))
	assert.Equal(t, "config.toml", filepath.Base(path))
}
