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
	require.NoError(t, scaffold.Run("", packageOptions(), discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(dir), m.Package.Name)
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(dir, "typst.toml"))
	assert.FileExists(t, filepath.Join(dir, "lib.typ"))
}

// TestRun_ZeroOptions_ScaffoldsPackage pins the default. A kind nobody set has
// to mean package, or init writes an empty directory and reports success.
func TestRun_ZeroOptions_ScaffoldsPackage(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("bare", scaffold.Options{}, discardLogger()))

	dir := filepath.Join(parent, "bare")
	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "lib.typ", m.Package.Entrypoint)

	assert.FileExists(t, filepath.Join(dir, "lib.typ"))
	assert.NoFileExists(t, filepath.Join(dir, "main.typ"))
}

func TestRun_WithArg_CreatesSubdir(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("my-pkg", packageOptions(), discardLogger()))

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
	require.NoError(t, scaffold.Run("", packageOptions(), discardLogger()))

	content, err := os.ReadFile(filepath.Join(dir, "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, string(scaffold.LibFile), string(content))
}

func TestRun_TomlContainsName(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("cool-pkg", packageOptions(), discardLogger()))

	raw, err := os.ReadFile(filepath.Join(parent, "cool-pkg", "typst.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"cool-pkg"`)
}

func TestRun_ExistingDirFails(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	// pre-create the subdir so Mkdir fails
	require.NoError(t, os.Mkdir(filepath.Join(parent, "dup"), paths.DirPerm))
	err := scaffold.Run("dup", packageOptions(), discardLogger())
	assert.Error(t, err)
}

// TestRun_Package_RecordsNoKind is the other half of the marker: a package is
// what an absent kind means, so writing one would be redundant.
func TestRun_Package_RecordsNoKind(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("plain", packageOptions(), discardLogger()))

	pkgDir := filepath.Join(parent, "plain")
	m, err := manifest.LoadFrom(pkgDir)
	require.NoError(t, err)
	assert.Empty(t, m.Tool.Gotpm.Kind)

	raw, err := os.ReadFile(filepath.Join(pkgDir, "typst.toml"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "[tool.gotpm]")
}

func TestRun_Document_WritesMainAndRecordsKind(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)
	require.NoError(t, scaffold.Run("thesis", documentOptions(), discardLogger()))

	docDir := filepath.Join(parent, "thesis")
	m, err := manifest.LoadFrom(docDir)
	require.NoError(t, err)
	assert.Equal(t, "thesis", m.Package.Name)
	// A document is never published and never imported, but the manifest is
	// only loadable at all with a version and an entrypoint in it.
	assert.Equal(t, "0.1.0", m.Package.Version)
	assert.Equal(t, "main.typ", m.Package.Entrypoint)
	assert.Equal(t, string(scaffold.KindDocument), m.Tool.Gotpm.Kind)

	assert.FileExists(t, filepath.Join(docDir, "main.typ"))
	assert.NoFileExists(t, filepath.Join(docDir, "lib.typ"))
}

func TestRun_Document_MainFileContent(t *testing.T) { //nolint: paralleltest
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, scaffold.Run("", documentOptions(), discardLogger()))

	content, err := os.ReadFile(filepath.Join(dir, "main.typ"))
	require.NoError(t, err)
	assert.Equal(t, string(scaffold.MainFile), string(content))
}

func TestRun_Document_NoArgs_UsesBasename(t *testing.T) { //nolint: paralleltest
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, scaffold.Run("", documentOptions(), discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(dir), m.Package.Name)
}

// TestRun_UnknownKind_WritesNothing keeps a kind gotpm does not know from
// scaffolding an empty directory and reporting success.
func TestRun_UnknownKind_WritesNothing(t *testing.T) { //nolint: paralleltest
	parent := t.TempDir()
	t.Chdir(parent)

	err := scaffold.Run("nope", scaffold.Options{Kind: scaffold.Kind("template")}, discardLogger())
	require.ErrorIs(t, err, scaffold.ErrUnknownKind)
	assert.NoDirExists(t, filepath.Join(parent, "nope"))
}

func packageOptions() scaffold.Options {
	return scaffold.Options{Kind: scaffold.KindPackage}
}

func documentOptions() scaffold.Options {
	return scaffold.Options{Kind: scaffold.KindDocument}
}
