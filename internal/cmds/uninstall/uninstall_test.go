package uninstall_test

import (
	"io"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/uninstall"
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

// installed creates a package directory containing the given versions of my-pkg in
// the local namespace, and returns the store root.
func installed(t *testing.T, versions ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, version := range versions {
		dir := filepath.Join(root, "local", "my-pkg", version)
		require.NoError(t, paths.EnsureDir(dir))
		require.NoError(t, paths.WriteFile(filepath.Join(dir, "lib.typ"), []byte(version)))
	}
	return root
}

// opts returns options pointing at root, which the store treats as the final
// destination, so a nested layout has to be built by the caller.
func opts(root string) *uninstall.Options {
	return &uninstall.Options{Namespace: paths.DefaultNamespace, InstallDir: root}
}

// storeOpts points the store at root without the flat override, so the
// namespace/name/version layout applies.
func storeOpts(t *testing.T, root string) *uninstall.Options {
	t.Helper()
	t.Setenv(paths.InstallDirEnvVar, "")
	t.Setenv("TYPST_PACKAGE_PATH", root)
	return &uninstall.Options{Namespace: paths.DefaultNamespace}
}

func TestRun_RemovesTheGivenVersion(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	o := storeOpts(t, root)
	o.Version = "0.1.0"

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.2.0"), "other versions must survive")
}

func TestRun_RemovesEveryVersionWithAll(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	o := storeOpts(t, root)
	o.All = true

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg"))
}

func TestRun_AllWithAVersionStaysScopedToIt(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	o := storeOpts(t, root)
	o.All = true
	o.Version = "0.1.0"

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.2.0"))
}

func TestRun_UsesTheManifestWithoutAName(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0")

	pkgDir := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(pkgDir, "typst.toml"), []byte(manifestTOML)))
	t.Chdir(pkgDir)

	require.NoError(t, uninstall.Run("", storeOpts(t, root), discardLogger()))
	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
}

func TestRun_NameWithoutVersionOrAllIsRejected(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0")

	err := uninstall.Run("my-pkg", storeOpts(t, root), discardLogger())
	require.ErrorIs(t, err, uninstall.ErrInsufficientPackage)
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
}

func TestRun_NotInstalled(t *testing.T) { //nolint: paralleltest
	root := installed(t)
	o := storeOpts(t, root)
	o.Version = "9.9.9"

	err := uninstall.Run("my-pkg", o, discardLogger())
	assert.ErrorIs(t, err, store.ErrNotInstalled)
}

func TestRun_NotInstalledWithAll(t *testing.T) { //nolint: paralleltest
	root := installed(t)
	o := storeOpts(t, root)
	o.All = true

	err := uninstall.Run("my-pkg", o, discardLogger())
	assert.ErrorIs(t, err, store.ErrNotInstalled)
}

func TestRun_DryRunKeepsThePackage(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0")
	o := storeOpts(t, root)
	o.Version = "0.1.0"
	o.DryRun = true

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
}

func TestRun_DryRunWithAllKeepsThePackage(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	o := storeOpts(t, root)
	o.All = true
	o.DryRun = true

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.2.0"))
}

func TestRun_InvalidVersionIsRejected(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0")
	o := storeOpts(t, root)
	o.Version = "not-a-version"

	err := uninstall.Run("my-pkg", o, discardLogger())
	require.Error(t, err)
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"))
}

func TestRun_InstallDirOverrideIsTheTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(root, "lib.typ"), []byte("x")))

	o := opts(root)
	o.Version = "0.1.0"
	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))

	assert.NoDirExists(t, root, "an overridden directory is removed as-is")
}

func TestRun_RespectsTheNamespace(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	for _, ns := range []string{"local", "preview"} {
		dir := filepath.Join(root, ns, "my-pkg", "0.1.0")
		require.NoError(t, paths.EnsureDir(dir))
	}
	o := storeOpts(t, root)
	o.Namespace = "preview"
	o.Version = "0.1.0"

	require.NoError(t, uninstall.Run("my-pkg", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0"))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"), "other namespaces must survive")
}

// installedIn creates the given versions of my-pkg in a namespace of root.
func installedIn(t *testing.T, root, namespace string, versions ...string) {
	t.Helper()
	for _, version := range versions {
		require.NoError(t, paths.EnsureDir(filepath.Join(root, namespace, "my-pkg", version)))
	}
}

// inPackageDir moves the test into a directory holding a typst.toml for
// my-pkg 0.1.0, which is what an uninstall without a name reads.
func inPackageDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "typst.toml"), []byte(manifestTOML)))
	t.Chdir(dir)
}

// namespaceOpts asks for the removal of a whole namespace: the namespace is
// named on the command line and nothing else is.
func namespaceOpts(t *testing.T, root, namespace string) *uninstall.Options {
	t.Helper()
	o := storeOpts(t, root)
	o.Namespace = namespace
	o.NamespaceSet = true
	return o
}

// answers returns a confirmation stub that gives the same answer every time,
// and a pointer to the number of times it was asked.
func answers(reply bool) (func(string) (bool, error), *int) {
	asked := 0
	return func(string) (bool, error) {
		asked++
		return reply, nil
	}, &asked
}

func TestRun_NamespaceAloneRemovesTheWholeNamespace(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0", "0.2.0")
	installedIn(t, root, "local", "0.1.0")

	o := namespaceOpts(t, root, "preview")
	confirm, asked := answers(true)
	o.Confirm = confirm

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.Equal(t, 1, *asked, "a namespace must not be deleted without being asked about")
	assert.NoDirExists(t, filepath.Join(root, "preview"))
	assert.DirExists(t, filepath.Join(root, "local"), "other namespaces must survive")
}

func TestRun_NamespaceIsKeptWhenTheAnswerIsNo(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0")

	o := namespaceOpts(t, root, "preview")
	confirm, _ := answers(false)
	o.Confirm = confirm

	require.NoError(t, uninstall.Run("", o, discardLogger()), "declining is an answer, not a failure")
	assert.DirExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0"))
}

func TestRun_YesSkipsTheQuestion(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0")

	o := namespaceOpts(t, root, "preview")
	o.Yes = true
	confirm, asked := answers(false)
	o.Confirm = confirm

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.Equal(t, 0, *asked, "--yes has already answered")
	assert.NoDirExists(t, filepath.Join(root, "preview"))
}

func TestRun_NamespaceNeedsSomeoneToConfirmIt(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0")

	// No stub and no terminal under test, so there is nobody to ask.
	err := uninstall.Run("", namespaceOpts(t, root, "preview"), discardLogger())

	require.ErrorIs(t, err, uninstall.ErrNeedsConfirmation)
	assert.DirExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0"))
}

func TestRun_DryRunKeepsTheNamespaceAndAsksNothing(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0")

	o := namespaceOpts(t, root, "preview")
	o.DryRun = true
	confirm, asked := answers(true)
	o.Confirm = confirm

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.Equal(t, 0, *asked, "a dry run deletes nothing, so there is nothing to confirm")
	assert.DirExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0"))
}

func TestRun_MissingNamespaceIsReported(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "local", "0.1.0")

	o := namespaceOpts(t, root, "nope")
	o.Yes = true

	assert.ErrorIs(t, uninstall.Run("", o, discardLogger()), store.ErrNotInstalled)
}

func TestRun_NamespaceCannotBeRemovedFromAnOverriddenDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(root, "lib.typ"), []byte("x")))

	o := opts(root)
	o.Namespace = "preview"
	o.NamespaceSet = true
	o.Yes = true

	require.ErrorIs(t, uninstall.Run("", o, discardLogger()), store.ErrFlatStore)
	assert.DirExists(t, root, "an overridden directory must survive a stray namespace flag")
}

func TestRun_NamespaceWithAVersionStaysScopedToThePackage(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0", "0.2.0")
	inPackageDir(t)

	o := namespaceOpts(t, root, "preview")
	o.Version = "0.2.0"

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "preview", "my-pkg", "0.2.0"),
		"a version on the command line wins over the one in the manifest")
	assert.DirExists(t, filepath.Join(root, "preview", "my-pkg", "0.1.0"))
}

func TestRun_NamespaceWithAllStaysScopedToThePackage(t *testing.T) { //nolint: paralleltest
	root := t.TempDir()
	installedIn(t, root, "preview", "0.1.0", "0.2.0")
	require.NoError(t, paths.EnsureDir(filepath.Join(root, "preview", "other-pkg", "0.1.0")))
	inPackageDir(t)

	o := namespaceOpts(t, root, "preview")
	o.All = true

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "preview", "my-pkg"))
	assert.DirExists(t, filepath.Join(root, "preview", "other-pkg", "0.1.0"),
		"--all names every version of the package, not every package of the namespace")
}

func TestRun_AllWithoutANameRemovesEveryVersionOfTheManifestPackage(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	inPackageDir(t)

	o := storeOpts(t, root)
	o.All = true

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg"))
}

func TestRun_VersionWithoutANameWinsOverTheManifest(t *testing.T) { //nolint: paralleltest
	root := installed(t, "0.1.0", "0.2.0")
	inPackageDir(t)

	o := storeOpts(t, root)
	o.Version = "0.2.0"

	require.NoError(t, uninstall.Run("", o, discardLogger()))

	assert.NoDirExists(t, filepath.Join(root, "local", "my-pkg", "0.2.0"))
	assert.DirExists(t, filepath.Join(root, "local", "my-pkg", "0.1.0"),
		"the manifest version must not be removed when another one was asked for")
}
