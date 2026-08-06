package typstsrc_test

import (
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/typstsrc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// write creates a file below dir and returns its path.
func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))
	require.NoError(t, paths.WriteFile(path, []byte(content)))
	return path
}

// statements reduces imports to what was written in the source.
func statements(imports []typstsrc.Import) []string {
	out := make([]string, 0, len(imports))
	for _, imp := range imports {
		out = append(out, imp.Statement)
	}
	return out
}

func TestScanFile_PackageAndFileImports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := write(t, dir, "main.typ", `#import "@preview/cetz:0.5.2"
#import "utils/helper.typ"
#let x = 1
`)

	imports, err := typstsrc.ScanFile(path)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	assert.True(t, imports[0].IsPackage())
	assert.Equal(t, "@preview/cetz:0.5.2", imports[0].Statement)
	assert.Empty(t, imports[0].File)

	assert.False(t, imports[1].IsPackage())
	assert.Equal(t, filepath.Join(dir, "utils", "helper.typ"), imports[1].File,
		"a relative import resolves against the importing file")
}

func TestScanFile_DescendsIntoIncludes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "chapter.typ", `#import "@preview/tablex:0.0.8"`)
	path := write(t, dir, "main.typ", `#import "@preview/cetz:0.5.2"
#include "chapter.typ"
`)

	imports, err := typstsrc.ScanFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"@preview/cetz:0.5.2", "@preview/tablex:0.0.8"}, statements(imports))
}

func TestScanFile_NestedIncludesResolveRelativeToTheirOwnFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "parts/deep.typ", `#import "@preview/deep:1.0.0"`)
	write(t, dir, "parts/chapter.typ", `#include "deep.typ"`)
	path := write(t, dir, "main.typ", `#include "parts/chapter.typ"`)

	imports, err := typstsrc.ScanFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"@preview/deep:1.0.0"}, statements(imports))
}

func TestScanFile_MissingIncludeIsAnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := write(t, dir, "main.typ", `#include "nope.typ"`)

	_, err := typstsrc.ScanFile(path)
	assert.ErrorContains(t, err, "could not open file")
}

func TestScanFile_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := typstsrc.ScanFile(filepath.Join(t.TempDir(), "nope.typ"))
	assert.ErrorContains(t, err, "could not open file")
}

func TestScanFile_IgnoresUnterminatedAndIndentedStatements(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := write(t, dir, "main.typ", `#import "@preview/unterminated
  #import "@preview/indented:1.0.0"
#import "@preview/good:1.0.0"
`)

	imports, err := typstsrc.ScanFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"@preview/good:1.0.0"}, statements(imports),
		"only statements starting a line and closing their quote are read")
}

func TestScanFile_NoImports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := write(t, dir, "main.typ", "#let x = 1\n")

	imports, err := typstsrc.ScanFile(path)
	require.NoError(t, err)
	assert.Empty(t, imports)
}
