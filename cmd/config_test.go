package cmd_test

import (
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/cmd"
	"github.com/npikall/gotpm/internal/config"
	"github.com/spf13/cobra"
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

	require.NoError(t, ConfigSetRunner(&cobra.Command{}, []string{"fork.path", "/tmp/my-fork"}))

	path, err := config.Path()
	require.NoError(t, err)
	assert.FileExists(t, path)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", cfg.Fork.Path)

	require.NoError(t, ConfigGetRunner(&cobra.Command{}, []string{"fork.path"}))

	require.NoError(t, ConfigListRunner(&cobra.Command{}, nil))

	require.NoError(t, ConfigUnsetRunner(&cobra.Command{}, []string{"fork.path"}))
	cfg, err = config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.Fork.Path)
}

func TestConfigSetRunner_UnknownKey(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	err := ConfigSetRunner(&cobra.Command{}, []string{"fork.nope", "value"})
	assert.ErrorIs(t, err, config.ErrUnknownKey)
}

func TestConfigGetRunner_UnknownKey(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	err := ConfigGetRunner(&cobra.Command{}, []string{"fork.nope"})
	assert.ErrorIs(t, err, config.ErrUnknownKey)
}

func TestConfigFile_NestedTable(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	require.NoError(t, ConfigSetRunner(&cobra.Command{}, []string{"fork.path", "/tmp/my-fork"}))

	path, err := config.Path()
	require.NoError(t, err)
	assert.Equal(t, "config.toml", filepath.Base(path))
}
