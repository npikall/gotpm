package lockfile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entry builds a lock entry with plausible provenance, so tests only have to
// spell out the fields they actually care about.
func entry(imp string, direct bool, requiredBy ...string) lockfile.Entry {
	name, version, _ := strings.Cut(strings.TrimPrefix(imp, "@gotpm/"), ":")
	return lockfile.Entry{
		Import:     imp,
		Name:       name,
		Version:    version,
		Namespace:  "gotpm",
		URL:        "github.com/example/" + name,
		Revision:   "v" + version,
		Hash:       "hash-" + name,
		Direct:     direct,
		RequiredBy: requiredBy,
	}
}

func imports(entries []lockfile.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Import)
	}
	return out
}

func TestLoad_MissingFileIsAnEmptyLock(t *testing.T) {
	t.Parallel()

	lock, err := lockfile.Load(t.TempDir())
	require.NoError(t, err, "a project without a lock file yet must not be an error")
	assert.Equal(t, lockfile.SchemaVersion, lock.Version)
	assert.Empty(t, lock.Packages)
}

func TestLoad_RejectsAFutureFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, paths.WriteFile(lockfile.Path(dir), []byte(`{"version":99,"packages":[]}`)))

	_, err := lockfile.Load(dir)
	require.ErrorIs(t, err, lockfile.ErrUnknownFormat)
}

func TestLoad_RejectsMalformedJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, paths.WriteFile(lockfile.Path(dir), []byte("not json")))

	_, err := lockfile.Load(dir)
	require.ErrorIs(t, err, lockfile.ErrInvalidLock)
}

func TestSave_RoundTrips(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/cetz:0.3.1", false, "@gotpm/mylib:1.0.0"))
	lock.Upsert(entry("@gotpm/mylib:1.0.0", true))
	require.NoError(t, lockfile.Save(dir, lock))

	loaded, err := lockfile.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, lock.Packages, loaded.Packages)
}

func TestSave_IsSortedAndNewlineTerminated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/zebra:1.0.0", true))
	lock.Upsert(entry("@gotpm/alpha:1.0.0", true))
	require.NoError(t, lockfile.Save(dir, lock))

	data, err := os.ReadFile(filepath.Join(dir, lockfile.FileName))
	require.NoError(t, err)
	text := string(data)

	assert.Less(t, strings.Index(text, "alpha"), strings.Index(text, "zebra"),
		"entries must be sorted by import so diffs stay stable")
	assert.True(t, strings.HasSuffix(text, "\n"))
}

func TestUpsert_MergesDependantsAndKeepsDirect(t *testing.T) {
	t.Parallel()

	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/cetz:0.3.1", true))
	lock.Upsert(entry("@gotpm/cetz:0.3.1", false, "@gotpm/mylib:1.0.0"))
	lock.Upsert(entry("@gotpm/cetz:0.3.1", false, "@gotpm/other:2.0.0", "@gotpm/mylib:1.0.0"))

	require.Len(t, lock.Packages, 1, "the import statement is the key of an entry")
	got, ok := lock.Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.True(t, got.Direct, "a package added directly stays direct")
	assert.Equal(t, []string{"@gotpm/mylib:1.0.0", "@gotpm/other:2.0.0"}, got.RequiredBy,
		"dependants accumulate, sorted and deduplicated")
}

func TestUpsert_UpdatesProvenance(t *testing.T) {
	t.Parallel()

	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/cetz:0.3.1", true))

	repinned := entry("@gotpm/cetz:0.3.1", true)
	repinned.Hash = "9f8e7d6"
	lock.Upsert(repinned)

	got, ok := lock.Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, "9f8e7d6", got.Hash)
}

func TestPrune_DropsOrphanedTransitiveDependencies(t *testing.T) {
	t.Parallel()

	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/mylib:1.0.0", true))
	lock.Upsert(entry("@gotpm/cetz:0.3.1", false, "@gotpm/mylib:1.0.0"))
	lock.Upsert(entry("@gotpm/oxifmt:0.2.1", false, "@gotpm/cetz:0.3.1"))

	removed := lock.Prune(nil)

	assert.Empty(t, lock.Packages, "removing the only root orphans the whole chain")
	assert.ElementsMatch(t,
		[]string{"@gotpm/mylib:1.0.0", "@gotpm/cetz:0.3.1", "@gotpm/oxifmt:0.2.1"},
		imports(removed))
}

func TestPrune_KeepsDependenciesWithASurvivingParent(t *testing.T) {
	t.Parallel()

	// Diamond: both direct packages pull in the same version of cetz.
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/mylib:1.0.0", true))
	lock.Upsert(entry("@gotpm/other:2.0.0", true))
	lock.Upsert(entry("@gotpm/cetz:0.3.1", false, "@gotpm/mylib:1.0.0", "@gotpm/other:2.0.0"))

	removed := lock.Prune([]string{"@gotpm/other:2.0.0"})

	assert.Equal(t, []string{"@gotpm/mylib:1.0.0"}, imports(removed))
	assert.ElementsMatch(t,
		[]string{"@gotpm/other:2.0.0", "@gotpm/cetz:0.3.1"},
		imports(lock.Packages),
		"cetz survives because its other dependant is still declared")

	cetz, ok := lock.Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.Equal(t, []string{"@gotpm/other:2.0.0"}, cetz.RequiredBy,
		"a surviving entry must not keep pointing at a dependant that is gone")
}

func TestPrune_RemarksDirectFromTheManifest(t *testing.T) {
	t.Parallel()

	// cetz was added directly, then also became a dependency of mylib. The
	// user drops it from typst.toml but keeps mylib, so it demotes to
	// transitive rather than disappearing.
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/mylib:1.0.0", true))
	lock.Upsert(entry("@gotpm/cetz:0.3.1", true, "@gotpm/mylib:1.0.0"))

	removed := lock.Prune([]string{"@gotpm/mylib:1.0.0"})

	assert.Empty(t, removed)
	cetz, ok := lock.Get("@gotpm/cetz:0.3.1")
	require.True(t, ok)
	assert.False(t, cetz.Direct)
	assert.Equal(t, []string{"@gotpm/mylib:1.0.0"}, lock.Direct())
}

func TestPrune_SurvivesACycle(t *testing.T) {
	t.Parallel()

	// A malformed or hand-edited lock can contain a cycle; pruning must
	// terminate rather than loop.
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/a:1.0.0", false, "@gotpm/b:1.0.0"))
	lock.Upsert(entry("@gotpm/b:1.0.0", false, "@gotpm/a:1.0.0"))

	removed := lock.Prune(nil)

	assert.Empty(t, lock.Packages, "a cycle nothing declares is still an orphan")
	assert.Len(t, removed, 2)
}

func TestPrune_IgnoresDirectImportsMissingFromTheLock(t *testing.T) {
	t.Parallel()

	// sync reports these separately; Prune must not invent an entry for them.
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/mylib:1.0.0", true))

	removed := lock.Prune([]string{"@gotpm/mylib:1.0.0", "@gotpm/unknown:1.0.0"})

	assert.Empty(t, removed)
	assert.Equal(t, []string{"@gotpm/mylib:1.0.0"}, imports(lock.Packages))
}
