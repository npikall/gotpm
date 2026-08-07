package deps_test

import (
	"io"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
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
