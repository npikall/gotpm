package config_test

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
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

// useEditor makes cmd the editor config edit opens; it is run with the file
// to edit as "$1".
func useEditor(t *testing.T, cmd string) {
	t.Helper()
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", cmd)
}

func TestEdit_SavesValidChanges(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	useEditor(t, `printf '[fork]\nurl = "https://github.com/user/packages"\n' >`)

	require.NoError(t, configcmd.Edit(t.Context()))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/packages", cfg.Fork.URL)
}

func TestEdit_StartsFromTemplate(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	useEditor(t, "true")

	require.NoError(t, configcmd.Edit(t.Context()))

	path, err := config.Path()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, config.Template, string(data))
}

func TestEdit_KeepsExistingFile(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	require.NoError(t, configcmd.Set("fork.path", "/tmp/my-fork"))
	useEditor(t, `grep -q /tmp/my-fork`)

	require.NoError(t, configcmd.Edit(t.Context()))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", cfg.Fork.Path)
}

func TestEdit_RejectsInvalidChanges(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	require.NoError(t, configcmd.Set("fork.path", "/tmp/my-fork"))
	useEditor(t, `printf '[fork]\nnope = "x"\n' >`)

	err := configcmd.Edit(t.Context())

	require.ErrorIs(t, err, config.ErrUnknownKey)
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my-fork", cfg.Fork.Path)
	assertNoStrayFiles(t)
}

func TestEdit_EditorFails(t *testing.T) { //nolint: paralleltest
	isolateConfigDir(t)
	useEditor(t, "exit 1 ;")

	require.Error(t, configcmd.Edit(t.Context()))
	assertNoStrayFiles(t)
}

func assertNoStrayFiles(t *testing.T) {
	t.Helper()
	path, err := config.Path()
	require.NoError(t, err)
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	for _, entry := range entries {
		assert.Equal(t, "config.toml", entry.Name())
	}
}

func TestEditor(t *testing.T) {
	t.Setenv("VISUAL", "nano")
	t.Setenv("EDITOR", "vim")
	assert.Equal(t, "nano", configcmd.Editor())

	t.Setenv("VISUAL", "")
	assert.Equal(t, "vim", configcmd.Editor())

	if runtime.GOOS != "windows" {
		t.Setenv("EDITOR", "")
		assert.Equal(t, "vi", configcmd.Editor())
	}
}

func TestRetry(t *testing.T) {
	t.Parallel()
	invalid := config.ErrUnknownKey
	yes := func(string) (bool, error) { return true, nil }
	no := func(string) (bool, error) { return false, nil }
	broken := func(string) (bool, error) { return true, io.ErrUnexpectedEOF }

	assert.True(t, configcmd.Retry(invalid, true, yes))
	assert.False(t, configcmd.Retry(invalid, true, no))
	assert.False(t, configcmd.Retry(invalid, true, broken))
	assert.False(t, configcmd.Retry(invalid, false, yes), "nobody to ask")
	assert.False(t, configcmd.Retry(configcmd.ErrEditorFailed, true, yes), "edit abandoned")
}
