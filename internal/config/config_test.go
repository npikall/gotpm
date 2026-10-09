package config_test

import (
	"os"
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

func TestSave_EditsTheFileInPlace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	path, err := Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`# my settings
[fork]
url = "https://github.com/user/packages" # the org fork
`), 0o600))

	cfg, err := Load()
	require.NoError(t, err)
	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))
	require.NoError(t, Save(cfg))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `# my settings
[fork]
url = "https://github.com/user/packages" # the org fork
path = "/tmp/my-fork"
`, string(data))

	require.NoError(t, cfg.Unset("fork.url"))
	require.NoError(t, Save(cfg))

	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `# my settings
[fork]
path = "/tmp/my-fork"
`, string(data), "an unset key is removed")
}

func TestSave_LeavesUnchangedValuesAsWritten(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	path, err := Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("[fork]\nurl = 'https://x' # literal\n"), 0o600))

	cfg, err := Load()
	require.NoError(t, err)
	require.NoError(t, cfg.Set("fork.path", "/p"))
	require.NoError(t, Save(cfg))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "[fork]\nurl = 'https://x' # literal\npath = \"/p\"\n", string(data))
}

func TestSave_RemovesATableLeftEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg := &Config{}
	require.NoError(t, cfg.Set("fork.path", "/tmp/my-fork"))
	require.NoError(t, Save(cfg))

	require.NoError(t, cfg.Unset("fork.path"))
	require.NoError(t, Save(cfg))

	path, err := Path()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, string(data))
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

func TestLoad_MalformedFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	path, err := Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("fork = ["), 0o600))

	_, err = Load()

	require.ErrorContains(t, err, path)
}

func TestParse(t *testing.T) { //nolint: paralleltest
	cfg, err := Parse([]byte("[fork]\nurl = \"https://github.com/user/packages\"\n"))
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/packages", cfg.Fork.URL)
}

func TestParse_UnknownKey(t *testing.T) { //nolint: paralleltest
	_, err := Parse([]byte("[fork]\nnope = \"x\"\n"))
	require.ErrorIs(t, err, ErrUnknownKey)
	assert.ErrorContains(t, err, "fork.nope")
}

func TestParse_InvalidTOML(t *testing.T) { //nolint: paralleltest
	_, err := Parse([]byte("fork = ["))
	require.Error(t, err)
}

func TestTemplate_ParsesToEmptyConfig(t *testing.T) { //nolint: paralleltest
	cfg, err := Parse([]byte(Template))
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
	for _, entry := range cfg.Entries() {
		assert.Contains(t, Template, "# "+entry.Key+" = ")
	}
}
