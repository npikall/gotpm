package sync_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/add"
	"github.com/npikall/gotpm/internal/cmds/sync"
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

func lockOf(t *testing.T, projectDir string) *lockfile.Lock {
	t.Helper()
	lock, err := lockfile.Load(projectDir)
	require.NoError(t, err)
	return lock
}

// declare writes the dependency list of a project's manifest.
func declare(t *testing.T, projectDir string, imports ...string) {
	t.Helper()
	require.NoError(t, manifest.SetDependencies(filepath.Join(projectDir, manifest.FileName), imports))
}

// emptyStore deletes everything installed, leaving the project files as they
// are. This is what a fresh checkout of a committed project looks like.
func emptyStore(t *testing.T, packages string) {
	t.Helper()
	require.NoError(t, os.RemoveAll(packages))
}

func TestRun_RestoresAFreshCheckout(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))
	emptyStore(t, packages)

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	assert.True(t, installed(t, packages, "root", "1.0.0"))
	assert.True(t, installed(t, packages, "leaf", "1.0.0"),
		"the lock is what makes a transitive dependency reproducible")
	assert.FileExists(t, filepath.Join(packages, "gotpm", "leaf", "1.0.0", "lib.typ"))
	assert.Len(t, lockOf(t, project).Packages, 2)
}

func TestRun_InstallsTheLockedCommitNotTheTag(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))
	pinned := dep.Hash()

	// The author re-tags v0.3.1 onto a later commit. The lock still wins.
	dep.Release()
	require.NotEqual(t, pinned, dep.Hash())
	emptyStore(t, packages)

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	content, err := os.ReadFile(filepath.Join(packages, "gotpm", "cetz", "0.3.1", "lib.typ"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "build = 1", "the commit the lock pins is what gets installed")

	entry, ok := lockOf(t, project).Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, pinned, entry.Hash, "sync never re-pins on its own")
}

func TestRun_IsANoOpWhenEverythingIsInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))
	before, err := os.ReadFile(filepath.Join(project, lockfile.FileName))
	require.NoError(t, err)

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	after, err := os.ReadFile(filepath.Join(project, lockfile.FileName))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestRun_PrunesWhatTheManifestNoLongerDeclares(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	kept := testrepo.New(t, "kept", "1.0.0").Release()
	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(kept.URL(), &add.Options{}, discardLogger()))

	// The user deleted the line from typst.toml by hand.
	declare(t, project, kept.Import())

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	lock := lockOf(t, project)
	require.Len(t, lock.Packages, 1, "the dependency and what only it needed both go")
	_, ok := lock.Get(kept.Import())
	assert.True(t, ok)
	assert.True(t, installed(t, packages, "root", "1.0.0"),
		"pruning the lock does not touch the store, which other projects share")
}

func TestRun_RejectsADeclaredDependencyWithNoLockEntry(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	declare(t, project, "@gotpm/cetz:0.3.1")

	err := sync.Run(&sync.Options{}, discardLogger())

	require.ErrorIs(t, err, sync.ErrUnknownSource)
	assert.Contains(t, err.Error(), "@gotpm/cetz:0.3.1")
	assert.Contains(t, err.Error(), "gotpm add", "only the user knows the repository")
}

func TestRun_RejectsADependencyOutsideTheGotpmNamespace(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	declare(t, project, "@preview/cetz:0.3.1")

	err := sync.Run(&sync.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrInvalidDependency)
}

func TestRun_FrozenRefusesToRewriteAStaleLock(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	dropped := testrepo.New(t, "dropped", "1.0.0").Release()
	kept := testrepo.New(t, "kept", "1.0.0").Release()
	require.NoError(t, add.Run(dropped.URL(), &add.Options{}, discardLogger()))
	require.NoError(t, add.Run(kept.URL(), &add.Options{}, discardLogger()))
	declare(t, project, kept.Import())
	before, err := os.ReadFile(filepath.Join(project, lockfile.FileName))
	require.NoError(t, err)

	err = sync.Run(&sync.Options{Frozen: true}, discardLogger())

	require.ErrorIs(t, err, sync.ErrLockOutOfDate)
	assert.Contains(t, err.Error(), dropped.Import())
	after, readErr := os.ReadFile(filepath.Join(project, lockfile.FileName))
	require.NoError(t, readErr)
	assert.Equal(t, string(before), string(after), "--frozen must not write")
}

func TestRun_FrozenAcceptsALockThatAgreesWithTheManifest(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	require.NoError(t, add.Run(dep.URL(), &add.Options{}, discardLogger()))
	emptyStore(t, packages)

	require.NoError(t, sync.Run(&sync.Options{Frozen: true}, discardLogger()))

	assert.True(t, installed(t, packages, "cetz", "0.3.1"),
		"--frozen forbids writing the lock, not installing from it")
}

func TestRun_WithoutAProject(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())

	err := sync.Run(&sync.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrManifestNotFound)
}

func TestRun_InstallsIntoThePackageDirectoryDespiteAnInstallDirOverride(t *testing.T) {
	packages := testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)
	require.NoError(t, add.Run(root.URL(), &add.Options{}, discardLogger()))
	emptyStore(t, packages)

	installDir := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, installDir)
	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	for _, name := range []string{"root", "leaf"} {
		assert.True(t, installed(t, packages, name, "1.0.0"),
			"%s belongs in the package directory: an install dir holds one package, not a lock", name)
	}
	entries, err := os.ReadDir(installDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "the install dir is not sync's destination")
}
