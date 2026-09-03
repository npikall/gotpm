package install_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/install"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/testrepo"
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

func TestRun_DropsAProvenanceFileFoundInTheSource(t *testing.T) {
	t.Parallel()
	// The fork was itself checked out from a previous gotpm-managed install,
	// so it carries a record of a repository and commit its files no longer
	// match.
	src := sourcePackage(t, "forked")
	require.NoError(t, paths.WriteFile(filepath.Join(src, store.ProvenanceFile), []byte(`{"url":"github.com/x/cetz"}`)))
	opts, dest := destination(t)

	require.NoError(t, install.Run(src, opts, discardLogger()))

	assert.NoFileExists(t, filepath.Join(dest, store.ProvenanceFile),
		"install <path> must never leave the destination looking like a repository-tracked install")
}

func TestRun_EditableIgnoresAProvenanceFileInTheSource(t *testing.T) {
	t.Parallel()
	src := sourcePackage(t, "forked")
	require.NoError(t, paths.WriteFile(filepath.Join(src, store.ProvenanceFile), []byte(`{"url":"github.com/x/cetz"}`)))
	opts, _ := destination(t)
	opts.Editable = true

	require.NoError(t, install.Run(src, opts, discardLogger()))

	s, err := store.Open(opts.InstallDir)
	require.NoError(t, err)
	ref, err := pkg.New(opts.Namespace, "my-pkg", "0.1.0")
	require.NoError(t, err)
	_, found, err := s.ReadProvenance(ref)
	require.NoError(t, err)
	assert.False(t, found, "an editable install must not report the linked source's own provenance file as its own")
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

// remoteOptions installs into the shared package directory testrepo.Isolate
// pointed at, the way a real "gotpm install -r" does: a graph does not fit in
// a flat --install-dir destination.
func remoteOptions(url string) *install.Options {
	return &install.Options{Namespace: paths.DefaultNamespace, Remote: url}
}

func TestRun_Remote_InstallsTheRootUnderTheGivenNamespace(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	root := testrepo.New(t, "root", "1.0.0").Release()

	require.NoError(t, install.Run("", remoteOptions(root.URL()), discardLogger()))

	assert.FileExists(t, filepath.Join(packages, paths.DefaultNamespace, "root", "1.0.0", "lib.typ"))
}

func TestRun_Remote_InstallsTransitiveDependenciesUnderGotpm(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)

	require.NoError(t, install.Run("", remoteOptions(root.URL()), discardLogger()))

	assert.FileExists(t, filepath.Join(packages, paths.DefaultNamespace, "root", "1.0.0", "lib.typ"))
	assert.FileExists(t, filepath.Join(packages, manifest.Namespace, "leaf", "1.0.0", "lib.typ"))
}

func TestRun_Remote_SkipsADependencyWithNoLock(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").ReleaseWith([]string{leaf.Import()}, nil)

	err := install.Run("", remoteOptions(root.URL()), discardLogger())

	require.NoError(t, err, "a dependency with no gotpm.lock at all is a warning, not a fatal error")
	assert.FileExists(t, filepath.Join(packages, paths.DefaultNamespace, "root", "1.0.0", "lib.typ"))
	assert.NoDirExists(t, filepath.Join(packages, manifest.Namespace, "leaf", "1.0.0"))
}

func TestRun_Remote_FailsOnADependencyWithAnIncompleteLock(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	known := testrepo.New(t, "known", "1.0.0").Release()
	forgotten := testrepo.New(t, "forgotten", "1.0.0").Release()
	lock := lockfile.New()
	lock.Upsert(known.LockEntry())
	root := testrepo.New(t, "root", "1.0.0").ReleaseWith([]string{known.Import(), forgotten.Import()}, lock)

	err := install.Run("", remoteOptions(root.URL()), discardLogger())

	require.ErrorIs(t, err, install.ErrUnresolvable)
	assert.NoDirExists(t, filepath.Join(packages, paths.DefaultNamespace, "root", "1.0.0"),
		"a still-unresolvable dependency fails the whole install, not just its own subtree")
}

func TestRun_Remote_RejectsACoordinateTakenByAnotherRepository(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, install.Run("", remoteOptions(mine.URL()), discardLogger()))

	err := install.Run("", remoteOptions(theirs.URL()), discardLogger())

	require.ErrorIs(t, err, deps.ErrSourceConflict)
}

func TestRun_Remote_ForceReplacesACoordinateTakenByAnotherRepository(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, install.Run("", remoteOptions(mine.URL()), discardLogger()))

	forced := remoteOptions(theirs.URL())
	forced.Force = true
	require.NoError(t, install.Run("", forced, discardLogger()))

	ref, err := pkg.New(paths.DefaultNamespace, "cetz", "0.3.1")
	require.NoError(t, err)
	prov, ok, err := store.At(packages).ReadProvenance(ref)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, theirs.URL(), prov.URL)
}

func TestRun_Remote_RefusesAGraphInAFlatStore(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	opts := remoteOptions(root.URL())
	opts.InstallDir = filepath.Join(t.TempDir(), "vendor")

	err := install.Run("", opts, discardLogger())

	require.ErrorIs(t, err, install.ErrGraphInFlatStore)
	require.ErrorContains(t, err, "1 dependencies")
	assert.NoDirExists(t, opts.InstallDir, "nothing is written once the graph is known not to fit")
}

func TestRun_Remote_RefusesAFlatStoreEvenWhenTheOnlyDependencyIsUnresolved(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").ReleaseWith([]string{leaf.Import()}, nil)
	opts := remoteOptions(root.URL())
	opts.InstallDir = filepath.Join(t.TempDir(), "vendor")

	err := install.Run("", opts, discardLogger())

	require.ErrorIs(t, err, install.ErrGraphInFlatStore,
		"root declares a dependency either way, whether or not it resolves")
	assert.NoDirExists(t, opts.InstallDir, "nothing is written once the graph is known not to fit")
}

func TestRun_Remote_AllowsASinglePackageInAFlatStore(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	root := testrepo.New(t, "root", "1.0.0").Release()
	opts := remoteOptions(root.URL())
	opts.InstallDir = filepath.Join(t.TempDir(), "vendor")

	require.NoError(t, install.Run("", opts, discardLogger()))

	assert.FileExists(t, filepath.Join(opts.InstallDir, "lib.typ"))
}
