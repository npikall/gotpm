package remove_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/add"
	"github.com/npikall/gotpm/internal/cmds/remove"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
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

func TestRun_DropsTheDependencyFromTheProject(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run("@gotpm/cetz:0.3.1", &remove.Options{}, discardLogger()))

	assert.Empty(t, declared(t, project))
	assert.Empty(t, lockOf(t, project).Packages)
	assert.True(t, installed(t, packages, "cetz", "0.3.1"),
		"the store is shared, so removing a dependency does not delete the files")
}

func TestRun_PruneDeletesTheFilesToo(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run("@gotpm/cetz:0.3.1", &remove.Options{Prune: true}, discardLogger()))

	assert.False(t, installed(t, packages, "cetz", "0.3.1"))
}

func TestRun_TakesTransitiveDependenciesWithIt(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run(root.Import(), &remove.Options{Prune: true}, discardLogger()))

	assert.Empty(t, lockOf(t, project).Packages, "an orphaned transitive dependency goes with it")
	assert.False(t, installed(t, packages, "root", "1.0.0"))
	assert.False(t, installed(t, packages, "leaf", "1.0.0"))
}

func TestRun_KeepsATransitiveDependencySomethingElseNeeds(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	shared := testrepo.New(t, "shared", "1.0.0").Release()
	left := testrepo.New(t, "left", "1.0.0").Release(shared)
	right := testrepo.New(t, "right", "1.0.0").Release(shared)
	require.NoError(t, add.Run(left.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(right.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run(left.Import(), &remove.Options{Prune: true}, discardLogger()))

	lock := lockOf(t, project)
	require.Len(t, lock.Packages, 2)
	entry, ok := lock.Get(shared.Import())
	require.True(t, ok)
	assert.Equal(t, []string{right.Import()}, entry.RequiredBy,
		"the dependant that is gone stops being recorded")
	assert.True(t, installed(t, packages, "shared", "1.0.0"))
	assert.False(t, installed(t, packages, "left", "1.0.0"))
}

func TestRun_LeavesOtherVersionsAlone(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	old := testrepo.New(t, "cetz", "0.3.1").Release()
	recent := testrepo.New(t, "cetz", "0.4.0").Release()
	require.NoError(t, add.Run(old.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(recent.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run("@gotpm/cetz:0.3.1", &remove.Options{Prune: true}, discardLogger()))

	assert.Equal(t, []string{"@gotpm/cetz:0.4.0"}, declared(t, project))
	assert.False(t, installed(t, packages, "cetz", "0.3.1"))
	assert.True(t, installed(t, packages, "cetz", "0.4.0"))
}

func TestRun_RejectsSomethingTheProjectDoesNotDeclare(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")

	err := remove.Run("@gotpm/cetz:0.3.1", &remove.Options{}, discardLogger())

	require.ErrorIs(t, err, remove.ErrNotDeclared)
	assert.Contains(t, err.Error(), manifest.FileName)
}

func TestRun_RejectsAnImportOutsideTheGotpmNamespace(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")

	err := remove.Run("@preview/cetz:0.3.1", &remove.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrInvalidDependency)
}

func TestRun_RejectsAnImportWithoutAVersion(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")

	err := remove.Run("@gotpm/cetz", &remove.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrInvalidDependency)
}

func TestRun_PrunesFromThePackageDirectoryDespiteAnInstallDirOverride(t *testing.T) {
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))

	installDir := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, installDir)
	require.NoError(t, remove.Run(root.Import(), &remove.Options{Prune: true}, discardLogger()))

	for _, name := range []string{"root", "leaf"} {
		assert.False(t, installed(t, packages, name, "1.0.0"),
			"%s must be pruned from the package directory it was installed into", name)
	}
}
