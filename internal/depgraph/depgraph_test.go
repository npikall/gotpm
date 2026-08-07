package depgraph_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/log/v2"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

// isolateDataDir points gotpm's data directory at a temporary one, so tests
// never touch the developer's real clone cache.
func isolateDataDir(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("HOME", data)
	t.Setenv("APPDATA", data)
}

// pkgRepo is a git repository holding one typst package. Walk reaches it over
// file://, so the whole path — clone, checkout, manifest, lock — is exercised
// without a network.
type pkgRepo struct {
	t       *testing.T
	dir     string
	repo    *git.Repository
	name    string
	version string
	hash    string
}

// newPkg creates an empty repository for a package. Nothing is committed until
// release is called, because a dependency has to exist before the package that
// depends on it can lock the commit it sits at.
func newPkg(t *testing.T, name, version string) *pkgRepo {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	require.NoError(t, paths.EnsureDir(dir))
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	return &pkgRepo{t: t, dir: dir, repo: repo, name: name, version: version}
}

func (p *pkgRepo) url() string { return "file://" + p.dir }

func (p *pkgRepo) imp() string { return "@" + manifest.Namespace + "/" + p.name + ":" + p.version }

func (p *pkgRepo) tag() string { return "v" + p.version }

// release commits the package, declaring the given dependencies and locking
// each of them to the commit it currently sits at.
func (p *pkgRepo) release(deps ...*pkgRepo) *pkgRepo {
	p.t.Helper()
	declared := make([]string, 0, len(deps))
	lock := lockfile.New()
	for _, dep := range deps {
		declared = append(declared, dep.imp())
		lock.Upsert(dep.lockEntry())
	}
	return p.releaseWith(declared, lock)
}

// lockEntry is how a dependant pins this package.
func (p *pkgRepo) lockEntry() lockfile.Entry {
	require.NotEmpty(p.t, p.hash, "%s must be released before something can depend on it", p.name)
	return lockfile.Entry{
		Import:    p.imp(),
		Name:      p.name,
		Version:   p.version,
		Namespace: manifest.Namespace,
		URL:       p.url(),
		Revision:  p.tag(),
		Hash:      p.hash,
	}
}

// releaseWith commits the package declaring exactly declared, shipping lock. A
// nil lock ships no gotpm.lock at all, which is what a package that forgot to
// commit one looks like.
func (p *pkgRepo) releaseWith(declared []string, lock *lockfile.Lock) *pkgRepo {
	t := p.t
	t.Helper()

	require.NoError(t, paths.WriteFile(filepath.Join(p.dir, manifest.FileName), p.manifest(declared)))
	require.NoError(t, paths.WriteFile(filepath.Join(p.dir, "lib.typ"), []byte("#let name = \""+p.name+"\"")))
	if lock != nil {
		require.NoError(t, lockfile.Save(p.dir, lock))
	}

	wt, err := p.repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))
	hash, err := wt.Commit("release "+p.version, &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = p.repo.CreateTag(p.tag(), hash, nil)
	require.NoError(t, err)

	p.hash = hash.String()
	return p
}

func (p *pkgRepo) manifest(declared []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "[package]\nname = %q\nversion = %q\nentrypoint = \"lib.typ\"\n", p.name, p.version)
	if len(declared) > 0 {
		b.WriteString("\n[tool.gotpm]\ndependencies = [\n")
		for _, dep := range declared {
			fmt.Fprintf(&b, "  %q,\n", dep)
		}
		b.WriteString("]\n")
	}
	return []byte(b.String())
}

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

func walk(t *testing.T, root *pkgRepo) []lockfile.Entry {
	t.Helper()
	entries, err := depgraph.Walk(resolve.Request{URL: root.url()}, discardLogger())
	require.NoError(t, err)
	return entries
}

func TestWalk_PackageWithoutDependencies(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	root := newPkg(t, "root", "1.0.0").release()

	entries := walk(t, root)

	require.Len(t, entries, 1)
	assert.Equal(t, lockfile.Entry{
		Import:    "@gotpm/root:1.0.0",
		Name:      "root",
		Version:   "1.0.0",
		Namespace: "gotpm",
		URL:       root.url(),
		Revision:  "v1.0.0",
		Hash:      root.hash,
		Direct:    true,
	}, entries[0])
}

func TestWalk_ChainOfThree(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	leaf := newPkg(t, "leaf", "1.0.0").release()
	middle := newPkg(t, "middle", "1.0.0").release(leaf)
	root := newPkg(t, "root", "1.0.0").release(middle)

	entries := walk(t, root)

	assert.Equal(t, []string{root.imp(), middle.imp(), leaf.imp()}, imports(entries),
		"the requested package comes first, then what it pulls in")
	assert.True(t, entries[0].Direct)
	assert.Empty(t, entries[0].RequiredBy)
	assert.False(t, entries[1].Direct)
	assert.Equal(t, []string{root.imp()}, entries[1].RequiredBy)
	assert.Equal(t, []string{middle.imp()}, entries[2].RequiredBy)
}

func TestWalk_TransitiveEntriesRecordTheTagTheirLockPinned(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	leaf := newPkg(t, "leaf", "2.3.4").release()
	root := newPkg(t, "root", "1.0.0").release(leaf)

	entry := entryFor(t, walk(t, root), leaf.imp())

	assert.Equal(t, leaf.hash, entry.Hash, "a dependency is pinned to the commit its dependant locked")
	assert.Equal(t, "v2.3.4", entry.Revision, "the tag survives, even though the commit is what is fetched")
	assert.Equal(t, leaf.url(), entry.URL)
}

func TestWalk_DiamondYieldsOneEntryWithBothDependants(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	shared := newPkg(t, "shared", "1.0.0").release()
	left := newPkg(t, "left", "1.0.0").release(shared)
	right := newPkg(t, "right", "1.0.0").release(shared)
	root := newPkg(t, "root", "1.0.0").release(left, right)

	entries := walk(t, root)

	require.Len(t, entries, 4, "the shared dependency is visited twice but recorded once")
	assert.Equal(t, []string{left.imp(), right.imp()}, entryFor(t, entries, shared.imp()).RequiredBy)
}

func TestWalk_SamePackageAtTwoVersionsCoexists(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	// Typst installs every version at its own path, so a dependency needed at
	// two versions is not a conflict to resolve: both get installed.
	old := newPkg(t, "shared", "1.0.0").release()
	recent := newPkg(t, "shared", "2.0.0").release()
	left := newPkg(t, "left", "1.0.0").release(old)
	right := newPkg(t, "right", "1.0.0").release(recent)
	root := newPkg(t, "root", "1.0.0").release(left, right)

	entries := walk(t, root)

	require.Len(t, entries, 5)
	assert.Equal(t, []string{left.imp()}, entryFor(t, entries, "@gotpm/shared:1.0.0").RequiredBy)
	assert.Equal(t, []string{right.imp()}, entryFor(t, entries, "@gotpm/shared:2.0.0").RequiredBy)
}

func TestWalk_DependencyWithoutALock(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	leaf := newPkg(t, "leaf", "1.0.0").release()
	middle := newPkg(t, "middle", "1.0.0").releaseWith([]string{leaf.imp()}, nil)
	root := newPkg(t, "root", "1.0.0").release(middle)

	_, err := depgraph.Walk(resolve.Request{URL: root.url()}, discardLogger())

	require.ErrorIs(t, err, depgraph.ErrUnresolvable)
	assert.Contains(t, err.Error(), leaf.imp(), "the error names what could not be resolved")
	assert.Contains(t, err.Error(), middle.imp(), "and the package that must fix it")
	assert.Contains(t, err.Error(), lockfile.FileName)
}

func TestWalk_DependencyLockMissingAnEntry(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	known := newPkg(t, "known", "1.0.0").release()
	forgotten := newPkg(t, "forgotten", "1.0.0").release()

	lock := lockfile.New()
	lock.Upsert(known.lockEntry())
	middle := newPkg(t, "middle", "1.0.0").releaseWith([]string{known.imp(), forgotten.imp()}, lock)
	root := newPkg(t, "root", "1.0.0").release(middle)

	_, err := depgraph.Walk(resolve.Request{URL: root.url()}, discardLogger())

	require.ErrorIs(t, err, depgraph.ErrUnresolvable)
	assert.Contains(t, err.Error(), forgotten.imp())
	assert.NotContains(t, err.Error(), known.imp())
}

func TestWalk_RejectsADependencyOutsideTheGotpmNamespace(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	middle := newPkg(t, "middle", "1.0.0").releaseWith([]string{"@preview/cetz:0.3.1"}, lockfile.New())
	root := newPkg(t, "root", "1.0.0").release(middle)

	_, err := depgraph.Walk(resolve.Request{URL: root.url()}, discardLogger())

	require.ErrorIs(t, err, manifest.ErrInvalidDependency)
	assert.Contains(t, err.Error(), middle.imp(), "the offending entry is reported with its package")
}

func TestWalk_RejectsAGraphThatIsTooDeep(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	// One level past the cap, built from the leaf up so each package can lock
	// the one below it.
	const levels = 33
	var below *pkgRepo
	for i := range levels {
		p := newPkg(t, fmt.Sprintf("p%02d", i), "1.0.0")
		if below == nil {
			below = p.release()
			continue
		}
		below = p.release(below)
	}

	_, err := depgraph.Walk(resolve.Request{URL: below.url()}, discardLogger())

	require.ErrorIs(t, err, depgraph.ErrTooDeep)
	assert.Contains(t, err.Error(), "@gotpm/p32:1.0.0", "the error names the chain that got too long")
}

func TestWalk_ReportsWhatPulledInAnUnreachableDependency(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	gone := newPkg(t, "gone", "1.0.0").release()
	lock := lockfile.New()
	entry := gone.lockEntry()
	entry.URL = "file://" + filepath.Join(t.TempDir(), "deleted")
	lock.Upsert(entry)
	middle := newPkg(t, "middle", "1.0.0").releaseWith([]string{gone.imp()}, lock)
	root := newPkg(t, "root", "1.0.0").release(middle)

	_, err := depgraph.Walk(resolve.Request{URL: root.url()}, discardLogger())

	require.Error(t, err)
	assert.Contains(t, err.Error(), gone.imp())
	assert.Contains(t, err.Error(), middle.imp(), "a failed clone must say why gotpm went there")
}
