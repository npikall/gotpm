package deps_test

import (
	"io"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

func installer(t *testing.T) deps.Installer {
	t.Helper()
	s, err := store.Open("")
	require.NoError(t, err)
	return deps.Installer{Store: s, Logger: discardLogger()}
}

func TestEnsure_InstallsWhatIsMissing(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	result, err := installer(t).Ensure(dep.LockEntry())

	require.NoError(t, err)
	assert.Equal(t, deps.Installed, result.Outcome)
	assert.Equal(t, "@gotpm/cetz:0.3.1", result.Ref.String())
	assert.Empty(t, result.DriftWarning())
}

func TestEnsure_LeavesThePinnedCommitAlone(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	i := installer(t)
	_, err := i.Ensure(dep.LockEntry())
	require.NoError(t, err)

	result, err := i.Ensure(dep.LockEntry())

	require.NoError(t, err)
	assert.Equal(t, deps.UpToDate, result.Outcome)
}

func TestEnsure_ReplacesAnotherCommitOfTheSameVersion(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	i := installer(t)
	_, err := i.Ensure(dep.LockEntry())
	require.NoError(t, err)

	dep.Release() // the author published a second commit as the same version
	result, err := i.Ensure(dep.LockEntry())

	require.NoError(t, err)
	assert.Equal(t, deps.Replaced, result.Outcome,
		"the store must end up holding the commit the lock names")
}

func TestEnsure_ReportsATagThatHasMoved(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	pinned := dep.LockEntry()

	// The author re-tagged v0.3.1 onto a later commit after the pin was made.
	dep.Release()
	result, err := installer(t).Ensure(pinned)

	require.NoError(t, err)
	assert.Equal(t, dep.Hash(), result.MovedTag)
	assert.Contains(t, result.DriftWarning(), deps.ShortHash(pinned.Hash))
	assert.Contains(t, result.DriftWarning(), "gotpm add")
}

func TestEnsure_RefusesACoordinateInstalledFromElsewhere(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	i := installer(t)
	_, err := i.Ensure(mine.LockEntry())
	require.NoError(t, err)

	_, err = i.Ensure(theirs.LockEntry())

	require.ErrorIs(t, err, deps.ErrSourceConflict)
}

func TestEnsureAll_InstallsEveryEntryInOrder(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	cetz := testrepo.New(t, "cetz", "0.3.1").Release()
	fletcher := testrepo.New(t, "fletcher", "0.5.0").Release()

	results, err := installer(t).EnsureAll([]lockfile.Entry{cetz.LockEntry(), fletcher.LockEntry()})

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "@gotpm/cetz:0.3.1", results[0].Ref.String())
	assert.Equal(t, "@gotpm/fletcher:0.5.0", results[1].Ref.String())
}

func TestEnsureAll_StopsAtTheFirstEntryItCannotInstall(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	cetz := testrepo.New(t, "cetz", "0.3.1").Release()
	missing := lockfile.Entry{
		Import: "@gotpm/gone:0.1.0", Namespace: "gotpm", Name: "gone", Version: "0.1.0",
		URL: "file://" + filepath.Join(t.TempDir(), "no-such-repository"), Hash: cetz.Hash(),
	}
	after := testrepo.New(t, "fletcher", "0.5.0").Release()

	results, err := installer(t).EnsureAll([]lockfile.Entry{cetz.LockEntry(), missing, after.LockEntry()})

	require.Error(t, err)
	assert.Nil(t, results, "a half-installed set is not reported on")
}

func TestOpenInstaller_UsesTheOverriddenPackageDirectory(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	override := t.TempDir()

	i, err := deps.OpenInstaller(override, false, discardLogger())

	require.NoError(t, err)
	assert.Equal(t, override, i.Store.Root())
}

func TestOpenInstaller_PassesForceToTheInstaller(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	mine := testrepo.New(t, "cetz", "0.3.1").Release()
	theirs := testrepo.New(t, "cetz", "0.3.1").Release()
	i, err := deps.OpenInstaller("", true, discardLogger())
	require.NoError(t, err)
	_, err = i.Ensure(mine.LockEntry())
	require.NoError(t, err)

	result, err := i.Ensure(theirs.LockEntry())

	require.NoError(t, err, "force replaces a coordinate held by another repository")
	assert.Equal(t, deps.Replaced, result.Outcome)
}

func TestEnsure_RefusesAPackageItDidNotInstall(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	s, err := store.Open("")
	require.NoError(t, err)
	ref, err := pkg.New("gotpm", "cetz", "0.3.1")
	require.NoError(t, err)
	require.NoError(t, s.Install(ref, dep.Dir()))

	_, err = deps.Installer{Store: s, Logger: discardLogger()}.Ensure(dep.LockEntry())

	require.ErrorIs(t, err, deps.ErrSourceConflict)
	assert.Contains(t, err.Error(), store.ProvenanceFile,
		"a directory without provenance is not gotpm's to overwrite")
}
