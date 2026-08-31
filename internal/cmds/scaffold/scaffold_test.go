package scaffold_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/scaffold"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

func TestRun_NoArgs_UsesBasename(t *testing.T) { //nolint: paralleltest
	// cwd basename becomes the package name
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, scaffold.Run("", testDefaultOptions(), discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(dir), m.Package.Name)
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(dir, "typst.toml"))
	assert.FileExists(t, filepath.Join(dir, "lib.typ"))
}

func TestRun_WithArg_CreatesSubdir(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("my-pkg", testDefaultOptions(), discardLogger()))

	pkgDir := filepath.Join(parent, "my-pkg")
	assert.DirExists(t, pkgDir)

	m, err := manifest.LoadFrom(pkgDir)
	require.NoError(t, err)
	assert.Equal(t, "my-pkg", m.Package.Name)
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(pkgDir, "lib.typ"))
}

func TestRun_LibFileContent(t *testing.T) { //nolint: paralleltest
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, scaffold.Run("", testDefaultOptions(), discardLogger()))

	content, err := os.ReadFile(filepath.Join(dir, "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, string(scaffold.LibFile), string(content))
}

func TestRun_TomlContainsName(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("cool-pkg", testDefaultOptions(), discardLogger()))

	raw, err := os.ReadFile(filepath.Join(parent, "cool-pkg", "typst.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"cool-pkg"`)
}

func TestRun_ExistingDirFails(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	// pre-create the subdir so Mkdir fails
	require.NoError(t, os.Mkdir(filepath.Join(parent, "dup"), paths.DirPerm))
	err := scaffold.Run("dup", testDefaultOptions(), discardLogger())
	assert.Error(t, err)
}

func TestRun_ErrorsWhenDocAndLibPassed(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)

	opts := testDefaultOptions()
	opts.Library = true
	opts.Document = true

	require.ErrorIs(t, scaffold.Run("err", opts, discardLogger()), scaffold.ErrMutuallyExclusiveOpts)
}

func testDefaultOptions() scaffold.Options {
	return scaffold.Options{
		Document: false,
		Library:  true,
	}
}
