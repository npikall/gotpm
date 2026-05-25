package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInitCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().CountP("verbose", "v", "")
	return cmd
}

func TestInitRunner_NoArgs_UsesBasename(t *testing.T) {
	// cwd basename becomes the package name
	dir := t.TempDir()
	chdirTo(t, dir)
	require.NoError(t, initRunner(newInitCmd(t), []string{}))

	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(dir), m.Package.Name)
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(dir, "typst.toml"))
	assert.FileExists(t, filepath.Join(dir, "lib.typ"))
}

func TestInitRunner_WithArg_CreatesSubdir(t *testing.T) {
	parent := t.TempDir()
	chdirTo(t, parent)
	require.NoError(t, initRunner(newInitCmd(t), []string{"my-pkg"}))

	pkgDir := filepath.Join(parent, "my-pkg")
	assert.DirExists(t, pkgDir)

	m, err := internal.LoadManifest(pkgDir)
	require.NoError(t, err)
	assert.Equal(t, "my-pkg", m.Package.Name)
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(pkgDir, "lib.typ"))
}

func TestInitRunner_LibFileContent(t *testing.T) {
	dir := t.TempDir()
	chdirTo(t, dir)
	require.NoError(t, initRunner(newInitCmd(t), []string{}))

	content, err := os.ReadFile(filepath.Join(dir, "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, string(libFile), string(content))
}

func TestInitRunner_TomlContainsName(t *testing.T) {
	parent := t.TempDir()
	chdirTo(t, parent)
	require.NoError(t, initRunner(newInitCmd(t), []string{"cool-pkg"}))

	raw, err := os.ReadFile(filepath.Join(parent, "cool-pkg", "typst.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"cool-pkg"`)
}

func TestInitRunner_ExistingDirFails(t *testing.T) {
	parent := t.TempDir()
	chdirTo(t, parent)
	// pre-create the subdir so Mkdir fails
	require.NoError(t, os.Mkdir(filepath.Join(parent, "dup"), 0o755))
	err := initRunner(newInitCmd(t), []string{"dup"})
	assert.Error(t, err)
}
