package add_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/add"
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

func discardLogger() *log.Logger { return log.New(io.Discard) }

// installed reports whether a package version is present in the store.
func installed(t *testing.T, root, name, version string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, manifest.Namespace, name, version))
	return err == nil
}

func provenance(t *testing.T, root, name, version string) store.Provenance {
	t.Helper()
	ref, err := pkg.New(manifest.Namespace, name, version)
	require.NoError(t, err)
	prov, ok, err := store.At(root).ReadProvenance(ref)
	require.NoError(t, err)
	require.True(t, ok, "%s/%s has no provenance file", name, version)
	return prov
}

func declared(t *testing.T, projectDir string) []string {
	t.Helper()
	m, err := manifest.LoadFile(filepath.Join(projectDir, manifest.FileName))
	require.NoError(t, err)
	return m.Dependencies()
}

func lockOf(t *testing.T, projectDir string) *lockfile.Lock {
	t.Helper()
	lock, err := lockfile.Load(projectDir)
	require.NoError(t, err)
	return lock
}

func TestRun_InstallsAndRecordsTheDependency(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))

	assert.True(t, installed(t, packages, "cetz", "0.3.1"), "the package lands under the gotpm namespace")
	assert.FileExists(t, filepath.Join(packages, "gotpm", "cetz", "0.3.1", "lib.typ"))
	assert.Equal(t, store.Provenance{URL: dep.URL(), Revision: "v0.3.1", Hash: dep.Hash()},
		provenance(t, packages, "cetz", "0.3.1"))

	assert.Equal(t, []string{"@gotpm/cetz:0.3.1"}, declared(t, project))

	entry, ok := lockOf(t, project).Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, dep.URL(), entry.URL)
	assert.Equal(t, dep.Hash(), entry.Hash)
	assert.True(t, entry.Direct)
}

func TestRun_KeepsTheRestOfTheManifest(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))

	content, err := os.ReadFile(filepath.Join(project, manifest.FileName))
	require.NoError(t, err)
	assert.Contains(t, string(content), `name = "my-doc"`)
	assert.Contains(t, string(content), "[tool.gotpm]")
	assert.Contains(t, string(content), `"@gotpm/cetz:0.3.1",`)
}

func TestRun_InstallsTransitiveDependencies(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	middle := testrepo.New(t, "middle", "1.0.0").Release(leaf)
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))

	for _, name := range []string{"root", "middle", "leaf"} {
		assert.True(t, installed(t, packages, name, "1.0.0"), "%s must be installed", name)
	}
	assert.Equal(t, []string{root.Import()}, declared(t, project),
		"only the package that was asked for is declared")

	lock := lockOf(t, project)
	require.Len(t, lock.Packages, 3)
	entry, ok := lock.Get(leaf.Import())
	require.True(t, ok)
	assert.False(t, entry.Direct)
	assert.Equal(t, []string{middle.Import()}, entry.RequiredBy)
}

func TestRun_AddsASecondVersionAlongsideTheFirst(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	old := testrepo.New(t, "cetz", "0.3.1").Release()
	recent := testrepo.New(t, "cetz", "0.4.0").Release()

	require.NoError(t, add.Run(old.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(recent.URL(), &add.Options{}, discardLogger()))

	assert.True(t, installed(t, packages, "cetz", "0.3.1"))
	assert.True(t, installed(t, packages, "cetz", "0.4.0"))
	assert.Equal(t, []string{"@gotpm/cetz:0.3.1", "@gotpm/cetz:0.4.0"}, declared(t, project),
		"an import statement naming the old version must keep working")
	assert.Len(t, lockOf(t, project).Packages, 2)
}

func TestRun_AddingTheSamePackageTwiceDeclaresItOnce(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))

	assert.Equal(t, []string{"@gotpm/cetz:0.3.1"}, declared(t, project))
}

func TestRun_RejectsACoordinateTakenByAnotherRepository(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(mine.URL(), &add.Options{}, discardLogger()))

	err := add.Run(theirs.URL(), &add.Options{}, discardLogger())

	require.ErrorIs(t, err, deps.ErrSourceConflict)
	assert.Contains(t, err.Error(), mine.URL(), "the error names what is installed")
	assert.Contains(t, err.Error(), theirs.URL(), "and what was asked for")
	assert.Equal(t, mine.URL(), provenance(t, packages, "cetz", "0.3.1").URL,
		"the installed package is left alone")
}

func TestRun_ForceReplacesACoordinateTakenByAnotherRepository(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(mine.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, add.Run(theirs.URL(), &add.Options{Force: true}, discardLogger()))

	assert.Equal(t, theirs.URL(), provenance(t, packages, "cetz", "0.3.1").URL)
}

func TestRun_HonoursAnExplicitRevision(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1")
	dep.Release()
	wanted := dep.Hash()
	dep.Release() // a second commit on the same version

	require.NoError(t, add.Run(dep.URL(), &add.Options{Revision: wanted}, discardLogger()))

	entry, ok := lockOf(t, project).Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, wanted, entry.Hash)
}

func TestRun_WithoutAProject(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	err := add.Run(dep.URL(), &add.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrManifestNotFound)
	assert.Contains(t, err.Error(), "gotpm init")
}

func TestRun_LeavesTheProjectAloneWhenAnInstallFails(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(mine.URL(), &add.Options{}, discardLogger()))

	require.Error(t, add.Run(theirs.URL(), &add.Options{}, discardLogger()))

	assert.Equal(t, []string{"@gotpm/cetz:0.3.1"}, declared(t, project))
	entry, ok := lockOf(t, project).Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, mine.URL(), entry.URL, "a failed add must not repoint the lock")
}

func TestRun_InstallsIntoThePackageDirectoryDespiteAnInstallDirOverride(t *testing.T) {
	packages := testrepo.Isolate(t)
	installDir := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, installDir)
	testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)

	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))

	for _, name := range []string{"root", "leaf"} {
		assert.True(t, installed(t, packages, name, "1.0.0"),
			"%s belongs in the package directory: an install dir holds one package, not a graph", name)
	}
	entries, err := os.ReadDir(installDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "the install dir is not add's destination")
}
