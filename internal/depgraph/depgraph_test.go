package depgraph_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

// imports lists the entries the way they were returned, so order is testable.
func imports(entries []lockfile.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Import)
	}
	return out
}

func entryFor(t *testing.T, entries []lockfile.Entry, imp string) lockfile.Entry {
	t.Helper()
	for _, entry := range entries {
		if entry.Import == imp {
			return entry
		}
	}
	require.FailNowf(t, "missing entry", "no entry for %q in %v", imp, imports(entries))
	return lockfile.Entry{}
}

func walk(t *testing.T, root *testrepo.Package) []lockfile.Entry {
	t.Helper()
	entries, unresolved, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())
	require.NoError(t, err)
	require.Empty(t, unresolved)
	return entries
}

func TestWalk_PackageWithoutDependencies(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	root := testrepo.New(t, "root", "1.0.0").Release()

	entries := walk(t, root)

	require.Len(t, entries, 1)
	assert.Equal(t, lockfile.Entry{
		Import:    "@gotpm/root:1.0.0",
		Name:      "root",
		Version:   "1.0.0",
		Namespace: "gotpm",
		URL:       root.URL(),
		Revision:  "v1.0.0",
		Hash:      root.Hash(),
		Direct:    true,
	}, entries[0])
}

func TestWalk_ChainOfThree(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	middle := testrepo.New(t, "middle", "1.0.0").Release(leaf)
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	entries := walk(t, root)

	assert.Equal(t, []string{root.Import(), middle.Import(), leaf.Import()}, imports(entries),
		"the requested package comes first, then what it pulls in")
	assert.True(t, entries[0].Direct)
	assert.Empty(t, entries[0].RequiredBy)
	assert.False(t, entries[1].Direct)
	assert.Equal(t, []string{root.Import()}, entries[1].RequiredBy)
	assert.Equal(t, []string{middle.Import()}, entries[2].RequiredBy)
}

func TestWalk_TransitiveEntriesRecordTheTagTheirLockPinned(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "2.3.4").Release()
	root := testrepo.New(t, "root", "1.0.0").Release(leaf)

	entry := entryFor(t, walk(t, root), leaf.Import())

	assert.Equal(t, leaf.Hash(), entry.Hash, "a dependency is pinned to the commit its dependant locked")
	assert.Equal(t, "v2.3.4", entry.Revision, "the tag survives, even though the commit is what is fetched")
	assert.Equal(t, leaf.URL(), entry.URL)
}

func TestWalk_DiamondYieldsOneEntryWithBothDependants(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	shared := testrepo.New(t, "shared", "1.0.0").Release()
	left := testrepo.New(t, "left", "1.0.0").Release(shared)
	right := testrepo.New(t, "right", "1.0.0").Release(shared)
	root := testrepo.New(t, "root", "1.0.0").Release(left, right)

	entries := walk(t, root)

	require.Len(t, entries, 4, "the shared dependency is visited twice but recorded once")
	assert.Equal(t, []string{left.Import(), right.Import()}, entryFor(t, entries, shared.Import()).RequiredBy)
}

func TestWalk_SamePackageAtTwoVersionsCoexists(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	// Typst installs every version at its own path, so a dependency needed at
	// two versions is not a conflict to resolve: both get installed.
	old := testrepo.New(t, "shared", "1.0.0").Release()
	recent := testrepo.New(t, "shared", "2.0.0").Release()
	left := testrepo.New(t, "left", "1.0.0").Release(old)
	right := testrepo.New(t, "right", "1.0.0").Release(recent)
	root := testrepo.New(t, "root", "1.0.0").Release(left, right)

	entries := walk(t, root)

	require.Len(t, entries, 5)
	assert.Equal(t, []string{left.Import()}, entryFor(t, entries, "@gotpm/shared:1.0.0").RequiredBy)
	assert.Equal(t, []string{right.Import()}, entryFor(t, entries, "@gotpm/shared:2.0.0").RequiredBy)
}

func TestWalk_DependencyWithoutALock(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").Release()
	middle := testrepo.New(t, "middle", "1.0.0").ReleaseWith([]string{leaf.Import()}, nil)
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	entries, unresolved, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())

	require.NoError(t, err, "an unresolvable dependency is reported, not fatal to the walk")
	assert.Equal(t, []string{root.Import(), middle.Import()}, imports(entries),
		"everything reachable up to the unresolved dependency is still walked")
	require.Len(t, unresolved, 1)
	assert.Equal(t, leaf.Import(), unresolved[0].Dependency, "names what could not be resolved")
	assert.Equal(t, middle.Import(), unresolved[0].RequiredBy, "and the package that must fix it")
	assert.Equal(t, depgraph.NoLockShipped, unresolved[0].Reason)
}

func TestWalk_DependencyLockMissingAnEntry(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	known := testrepo.New(t, "known", "1.0.0").Release()
	forgotten := testrepo.New(t, "forgotten", "1.0.0").Release()

	lock := lockfile.New()
	lock.Upsert(known.LockEntry())
	middle := testrepo.New(t, "middle", "1.0.0").
		ReleaseWith([]string{known.Import(), forgotten.Import()}, lock)
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	entries, unresolved, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())

	require.NoError(t, err)
	assert.Equal(t, []string{root.Import(), middle.Import(), known.Import()}, imports(entries),
		"the declared dependency that does resolve is still installed")
	require.Len(t, unresolved, 1)
	assert.Equal(t, forgotten.Import(), unresolved[0].Dependency)
	assert.Equal(t, depgraph.IncompleteLock, unresolved[0].Reason)
}

func TestWalk_RejectsADependencyOutsideTheGotpmNamespace(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	middle := testrepo.New(t, "middle", "1.0.0").
		ReleaseWith([]string{"@preview/cetz:0.3.1"}, lockfile.New())
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	_, _, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrInvalidDependency)
	assert.Contains(t, err.Error(), middle.Import(), "the offending entry is reported with its package")
}

func TestWalk_RejectsAGraphThatIsTooDeep(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	// One level past the cap, built from the leaf up so each package can lock
	// the one below it.
	const levels = 33
	var below *testrepo.Package
	for i := range levels {
		p := testrepo.New(t, fmt.Sprintf("p%02d", i), "1.0.0")
		if below == nil {
			below = p.Release()
			continue
		}
		below = p.Release(below)
	}

	_, _, err := depgraph.Walk(resolve.Request{URL: below.URL()}, depgraph.Options{}, discardLogger())

	require.ErrorIs(t, err, depgraph.ErrTooDeep)
	assert.Contains(t, err.Error(), "@gotpm/p32:1.0.0", "the error names the chain that got too long")
}

func TestWalk_ReportsWhatPulledInAnUnreachableDependency(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	gone := testrepo.New(t, "gone", "1.0.0").Release()
	lock := lockfile.New()
	entry := gone.LockEntry()
	entry.URL = "file://" + filepath.Join(t.TempDir(), "deleted")
	lock.Upsert(entry)
	middle := testrepo.New(t, "middle", "1.0.0").ReleaseWith([]string{gone.Import()}, lock)
	root := testrepo.New(t, "root", "1.0.0").Release(middle)

	_, _, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())

	require.Error(t, err)
	assert.Contains(t, err.Error(), gone.Import())
	assert.Contains(t, err.Error(), middle.Import(), "a failed clone must say why gotpm went there")
}
