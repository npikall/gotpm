package update_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/update"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

// isolate points the data directories at a temp tree and seeds the index cache,
// so updating resolves against known data instead of the network.
func isolate(t *testing.T, idx index.Index) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("APPDATA", filepath.Join(tmp, "appdata"))
	require.NoError(t, index.SaveCache(idx))
}

// write creates a file below dir and returns its path.
func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))
	require.NoError(t, paths.WriteFile(path, []byte(content)))
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func defaultOpts() *update.Options {
	return &update.Options{Extensions: []string{".typ"}}
}

func TestRun_UpdatesAFileInPlace(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/cetz:0.1.0"`)

	require.NoError(t, update.Run([]string{file}, defaultOpts(), discardLogger()))
	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, file))
}

func TestRun_LeavesUpToDateImportsAlone(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/cetz:0.5.2"`)

	require.NoError(t, update.Run([]string{file}, defaultOpts(), discardLogger()))
	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, file))
}

func TestRun_WritesToTheOutputFile(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@preview/cetz:0.1.0"`)
	out := filepath.Join(dir, "out.typ")

	opts := defaultOpts()
	opts.Output = out
	require.NoError(t, update.Run([]string{file}, opts, discardLogger()))

	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, out))
	assert.Equal(t, `#import "@preview/cetz:0.1.0"`, read(t, file), "the input file stays untouched")
}

func TestRun_DirectoryProcessesItsFiles(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	top := write(t, dir, "a.typ", `#import "@preview/cetz:0.1.0"`)
	nested := write(t, dir, "sub/b.typ", `#import "@preview/cetz:0.1.0"`)
	other := write(t, dir, "notes.md", `#import "@preview/cetz:0.1.0"`)

	require.NoError(t, update.Run([]string{dir}, defaultOpts(), discardLogger()))

	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, top))
	assert.Equal(t, `#import "@preview/cetz:0.1.0"`, read(t, nested), "sub-directories need --recursive")
	assert.Equal(t, `#import "@preview/cetz:0.1.0"`, read(t, other), "only the given extensions are processed")
}

func TestRun_RecursiveDirectory(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	nested := write(t, dir, "sub/b.typ", `#import "@preview/cetz:0.1.0"`)

	opts := defaultOpts()
	opts.Recursive = true
	require.NoError(t, update.Run([]string{dir}, opts, discardLogger()))

	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, nested))
}

func TestRun_CustomExtensions(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	md := write(t, dir, "notes.md", `#import "@preview/cetz:0.1.0"`)

	opts := defaultOpts()
	opts.Extensions = []string{"md"} // a leading dot is optional
	require.NoError(t, update.Run([]string{dir}, opts, discardLogger()))

	assert.Equal(t, `#import "@preview/cetz:0.5.2"`, read(t, md))
}

func TestRun_OutputWithMultipleFilesIsRejected(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{"cetz": "0.5.2"})
	dir := t.TempDir()
	write(t, dir, "a.typ", `#import "@preview/cetz:0.1.0"`)
	write(t, dir, "b.typ", `#import "@preview/cetz:0.1.0"`)

	opts := defaultOpts()
	opts.Output = filepath.Join(dir, "out.typ")
	err := update.Run([]string{dir}, opts, discardLogger())
	assert.ErrorIs(t, err, update.ErrInvalidOutputOption)
}

func TestRun_NoInput(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})
	err := update.Run(nil, defaultOpts(), discardLogger())
	assert.ErrorIs(t, err, update.ErrMissingInput)
}

func TestRun_MissingInputFile(t *testing.T) { //nolint: paralleltest
	isolate(t, index.Index{})
	err := update.Run([]string{filepath.Join(t.TempDir(), "nope.typ")}, defaultOpts(), discardLogger())
	assert.ErrorContains(t, err, "could not read fileinfo")
}

func TestRun_UnknownPackageLeavesTheFileAlone(t *testing.T) { //nolint: paralleltest
	// An empty index sends the lookup to the GitHub API, which is unreachable
	// here; the failure must not corrupt the file.
	isolate(t, index.Index{})
	dir := t.TempDir()
	file := write(t, dir, "main.typ", `#import "@local/mine:0.1.0"`)

	require.NoError(t, update.Run([]string{file}, defaultOpts(), discardLogger()))
	assert.Equal(t, `#import "@local/mine:0.1.0"`, read(t, file), "other namespaces are not touched")
}
