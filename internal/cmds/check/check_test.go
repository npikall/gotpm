package check_test

import (
	"io"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/check"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

// isolate points the data directories at a temp tree and seeds the index cache,
// so checking resolves against known data instead of the network.
func isolate(t *testing.T, idx index.Index) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("APPDATA", filepath.Join(tmp, "appdata"))

	packagesRoot := filepath.Join(tmp, "packages")
	t.Setenv("TYPST_PACKAGE_PATH", packagesRoot)
	require.NoError(t, index.SaveCache(idx))
	return packagesRoot
}

// write creates a file below dir and returns its path.
func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))
	require.NoError(t, paths.WriteFile(path, []byte(content)))
	return path
}

func TestAnalyze_AllGood(t *testing.T) { //nolint: paralleltest
	packagesRoot := isolate(t, index.Index{"cetz": "0.5.2"})
	require.NoError(t, paths.EnsureDir(filepath.Join(packagesRoot, "local", "mine", "0.1.0")))

	dir := t.TempDir()
	write(t, dir, "helper.typ", "")
	file := write(t, dir, "main.typ", `#import "@preview/cetz:0.5.2"
#import "@local/mine:0.1.0"
#import "helper.typ"
`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	assert.Empty(t, result.Issues)
	assert.Empty(t, result.Warnings)
}

func TestAnalyze_OutdatedUniversePackageWarns(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/cetz:0.1.0"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	assert.Empty(t, result.Issues)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, check.Warning{Package: "cetz", Requested: "0.1.0", Latest: "0.5.2"}, result.Warnings[0])
}

func TestAnalyze_UnknownUniversePackage(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/nope:0.1.0"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	require.Len(t, result.Issues, 1)
	assert.ErrorIs(t, result.Issues[0].Err, check.ErrPackageNotInIndex)
}

func TestAnalyze_MissingLocalPackage(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@local/missing:0.1.0"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	require.Len(t, result.Issues, 1)
	assert.ErrorIs(t, result.Issues[0].Err, check.ErrPackageNotFound)
}

func TestAnalyze_MissingFile(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "nope.typ"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	require.Len(t, result.Issues, 1)
	assert.ErrorIs(t, result.Issues[0].Err, check.ErrFileNotFound)
}

func TestAnalyze_MalformedPackageImport(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/cetz"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	require.Len(t, result.Issues, 1)
	assert.Equal(t, "@preview/cetz", result.Issues[0].Import)
}

func TestAnalyze_ReportsEachImportOnce(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@local/missing:0.1.0"
#import "@local/missing:0.1.0"
`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	assert.Len(t, result.Issues, 1, "a repeated import is checked once")
}

func TestAnalyze_ChecksIncludedFilesToo(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	dir := t.TempDir()
	write(t, dir, "chapter.typ", `#import "@local/missing:0.1.0"`)
	file := write(t, dir, "main.typ", `#include "chapter.typ"`)

	result, err := check.Analyze(file, discardLogger())
	require.NoError(t, err)
	require.Len(t, result.Issues, 1)
	assert.Equal(t, "@local/missing:0.1.0", result.Issues[0].Import)
}

func TestAnalyze_UnreadableFile(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})

	_, err := check.Analyze(filepath.Join(t.TempDir(), "nope.typ"), discardLogger())
	assert.ErrorContains(t, err, "could not open file")
}
