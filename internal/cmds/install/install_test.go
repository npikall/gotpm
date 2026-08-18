package install_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/install"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const manifestTOML = `[package]
name = "my-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
`

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

// sourcePackage writes a package with one file and returns its directory.
func sourcePackage(t *testing.T, libContent string) string {
	t.Helper()
	dir := t.TempDir()
	// t.TempDir may sit behind a symlink, which would make the symlink
	// assertions compare two spellings of the same path.
	dir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "typst.toml"), []byte(manifestTOML)))
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "lib.typ"), []byte(libContent)))
	return dir
}

// destination returns options installing into a fresh directory, plus that
// directory. An install dir override makes it the destination itself.
func destination(t *testing.T) (*install.Options, string) {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "pkg-dest")
	return &install.Options{Namespace: paths.DefaultNamespace, InstallDir: dest}, dest
}

func TestRun_CopiesThePackage(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "v1")
	opts, dest := destination(t)

	require.NoError(t, install.Run(src, opts, discardLogger()))

	content, err := os.ReadFile(filepath.Join(dest, "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, "v1", string(content))
}

func TestRun_UsesTheWorkingDirectoryWithoutAPath(t *testing.T) { //nolint: paralleltest
	src := sourcePackage(t, "cwd")
	t.Chdir(src)
	opts, dest := destination(t)

	require.NoError(t, install.Run("", opts, discardLogger()))
	assert.FileExists(t, filepath.Join(dest, "lib.typ"))
}

func TestRun_FindsThePackageRootFromASubDirectory(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "root")
	nested := filepath.Join(src, "src", "deep")
	require.NoError(t, paths.EnsureDir(nested))
	opts, dest := destination(t)

	require.NoError(t, install.Run(nested, opts, discardLogger()))

	assert.FileExists(t, filepath.Join(dest, "typst.toml"))
	assert.FileExists(t, filepath.Join(dest, "lib.typ"), "the package root must be installed, not the sub-directory")
}

func TestRun_RejectsAnExistingInstall(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "v1")
	opts, _ := destination(t)

	require.NoError(t, install.Run(src, opts, discardLogger()))

	err := install.Run(src, opts, discardLogger())
	assert.ErrorIs(t, err, store.ErrAlreadyInstalled)
}

func TestRun_ForceOverwritesACopyInstall(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "v1")
	opts, dest := destination(t)
	require.NoError(t, install.Run(src, opts, discardLogger()))

	require.NoError(t, paths.WriteFile(filepath.Join(src, "lib.typ"), []byte("v2")))
	require.NoError(t, paths.WriteFile(filepath.Join(src, "extra.typ"), []byte("new")))

	opts.Force = true
	require.NoError(t, install.Run(src, opts, discardLogger()))

	content, err := os.ReadFile(filepath.Join(dest, "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, "v2", string(content))
	assert.FileExists(t, filepath.Join(dest, "extra.typ"))
}

func TestRun_Editable(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "content")
	opts, dest := destination(t)
	opts.Editable = true

	require.NoError(t, install.Run(src, opts, discardLogger()))

	info, err := os.Lstat(dest)
	require.NoError(t, err)
	assert.NotEqual(t, os.FileMode(0), info.Mode()&os.ModeSymlink)

	target, err := os.Readlink(dest)
	require.NoError(t, err)
	assert.Equal(t, src, target)
}

func TestRun_RemoteRejectsEditable(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "content")
	opts, _ := destination(t)
	opts.Editable = true
	opts.Remote = "github.com/npikall/gotpm"

	gotErr := install.Run(src, opts, discardLogger())
	require.ErrorIs(t, gotErr, install.ErrNoEditableRemote)
}

func TestRun_ForceReplacesAnEditableInstall(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "content")
	opts, dest := destination(t)
	opts.Editable = true
	require.NoError(t, install.Run(src, opts, discardLogger()))

	opts.Force = true
	require.NoError(t, install.Run(src, opts, discardLogger()))

	info, err := os.Lstat(dest)
	require.NoError(t, err)
	assert.NotEqual(t, os.FileMode(0), info.Mode()&os.ModeSymlink)
	target, err := os.Readlink(dest)
	require.NoError(t, err)
	assert.Equal(t, src, target)
}

func TestRun_InstallsIntoTheRequestedNamespace(t *testing.T) {
	src := sourcePackage(t, "content")
	root := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, "")
	t.Setenv("TYPST_PACKAGE_PATH", root)

	opts := &install.Options{Namespace: "preview"}
	require.NoError(t, install.Run(src, opts, discardLogger()))

	assert.FileExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0", "lib.typ"))
}

func TestRun_RejectsAnEmptyNamespace(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "content")
	opts, _ := destination(t)
	opts.Namespace = ""

	assert.Error(t, install.Run(src, opts, discardLogger()))
}

func TestRun_HonoursIgnoreFiles(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "content")
	require.NoError(t, paths.WriteFile(filepath.Join(src, "out.pdf"), []byte("binary")))
	require.NoError(t, paths.WriteFile(filepath.Join(src, ".typstignore"), []byte("*.pdf\n")))
	opts, dest := destination(t)

	require.NoError(t, install.Run(src, opts, discardLogger()))

	assert.FileExists(t, filepath.Join(dest, "lib.typ"))
	assert.NoFileExists(t, filepath.Join(dest, "out.pdf"))
}

func TestRun_PathErrors(t *testing.T) {
	t.Parallel()
	t.Run("path is a file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		file := filepath.Join(dir, "lib.typ")
		require.NoError(t, paths.WriteFile(file, []byte("")))
		opts, _ := destination(t)

		err := install.Run(file, opts, discardLogger())
		assert.ErrorIs(t, err, paths.ErrNotADirectory)
	})
	t.Run("path does not exist", func(t *testing.T) {
		t.Parallel()
		opts, _ := destination(t)
		err := install.Run(filepath.Join(t.TempDir(), "nope"), opts, discardLogger())
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
	t.Run("directory without a manifest", func(t *testing.T) {
		t.Parallel()
		opts, _ := destination(t)
		err := install.Run(t.TempDir(), opts, discardLogger())
		assert.ErrorContains(t, err, "could not load manifest")
	})
}

func TestRun_ResolvesARelativePath(t *testing.T) { //nolint: paralleltest
	src := sourcePackage(t, "content")
	t.Chdir(filepath.Dir(src))
	opts, dest := destination(t)

	require.NoError(t, install.Run(filepath.Base(src), opts, discardLogger()))
	assert.FileExists(t, filepath.Join(dest, "lib.typ"))
}
