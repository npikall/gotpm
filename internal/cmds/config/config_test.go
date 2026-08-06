package config_test

import (
	"path/filepath"
	"testing"

	configcmd "github.com/npikall/gotpm/internal/cmds/config"
	"github.com/npikall/gotpm/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isolateConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestConfigSetGetUnsetList(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)

	require.NoError(t, configcmd.Set("fork.path", "/tmp/my-fork"))

	path, err := config.Path()
	require.NoError(t, err)
	assert.FileExists(t, path)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", cfg.Fork.Path)

	require.NoError(t, configcmd.Get("fork.path"))

	require.NoError(t, configcmd.List())

	require.NoError(t, configcmd.Unset("fork.path"))
	cfg, err = config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.Fork.Path)
}

func TestSet_UnknownKey(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	err := configcmd.Set("fork.nope", "value")
	assert.ErrorIs(t, err, config.ErrUnknownKey)
}

func TestGet_UnknownKey(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	err := configcmd.Get("fork.nope")
	assert.ErrorIs(t, err, config.ErrUnknownKey)
}

func TestConfigFile_NestedTable(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	require.NoError(t, configcmd.Set("fork.path", "/tmp/my-fork"))

	path, err := config.Path()
	require.NoError(t, err)
	assert.Equal(t, "config.toml", filepath.Base(path))
}
