package resolve_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"charm.land/log/v2"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testURL = "github.com/example/my-pkg"

func discardLogger() *log.Logger { return log.New(io.Discard) }

// fixture is a repository sitting where gotpm caches its clones, so Resolve
// finds it without going near the network. Everything after the clone — tag
// selection, checkout and reading the manifest — is exercised for real.
type fixture struct {
	dir  string
	repo *git.Repository
}

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

func newRepo(t *testing.T, dir string) *fixture {
	t.Helper()
	require.NoError(t, paths.EnsureDir(dir))
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	return &fixture{dir: dir, repo: repo}
}

// seedCachedRepo puts a repository where gotpm caches its clones, so Resolve
// finds it without going near the network.
func seedCachedRepo(t *testing.T) *fixture {
	t.Helper()
	isolateDataDir(t)

	dir, err := remote.CachePath(testURL)
	require.NoError(t, err)
	return newRepo(t, dir)
}

// release commits a package at a version and returns the commit.
func (f *fixture) release(t *testing.T, version string) plumbing.Hash {
	t.Helper()
	f.writeManifest(t, version)
	require.NoError(t, paths.WriteFile(filepath.Join(f.dir, "lib.typ"), []byte("#let v = \""+version+"\"")))

	wt, err := f.repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))

	hash, err := wt.Commit("release "+version, &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)
	return hash
}

func (f *fixture) writeManifest(t *testing.T, version string) {
	t.Helper()
	content := `[package]
name = "my-pkg"
version = "` + version + `"
entrypoint = "lib.typ"
`
	require.NoError(t, paths.WriteFile(filepath.Join(f.dir, manifest.FileName), []byte(content)))
}

func (f *fixture) tag(t *testing.T, name string, hash plumbing.Hash) {
	t.Helper()
	_, err := f.repo.CreateTag(name, hash, nil)
	require.NoError(t, err)
}

func TestResolve_PicksTheNewestStableTag(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	f.tag(t, "v0.1.0", f.release(t, "0.1.0"))
	wanted := f.release(t, "0.2.0")
	f.tag(t, "v0.2.0", wanted)
	f.tag(t, "v0.3.0-rc1", f.release(t, "0.3.0"))
	f.tag(t, "nightly", f.release(t, "0.4.0"))

	got, err := resolve.Resolve(resolve.Request{URL: testURL}, discardLogger())
	require.NoError(t, err)

	assert.Equal(t, "v0.2.0", got.Revision, "a prerelease and a moving tag are not releases")
	assert.Equal(t, wanted.String(), got.Hash)
	assert.Equal(t, "0.2.0", got.Manifest.Package.Version)
	assert.Equal(t, "github.com/example/my-pkg", got.Source.Canonical)
}

func TestResolve_LatestMeansTheSameAsAskingForNothing(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	f.tag(t, "v0.1.0", f.release(t, "0.1.0"))
	f.tag(t, "v0.2.0", f.release(t, "0.2.0"))

	got, err := resolve.Resolve(resolve.Request{URL: testURL, Revision: resolve.Latest}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", got.Revision)
}

func TestResolve_HonoursAnExplicitTag(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	wanted := f.release(t, "0.1.0")
	f.tag(t, "v0.1.0", wanted)
	f.tag(t, "v0.2.0", f.release(t, "0.2.0"))

	got, err := resolve.Resolve(resolve.Request{URL: testURL, Revision: "v0.1.0"}, discardLogger())
	require.NoError(t, err)

	assert.Equal(t, "v0.1.0", got.Revision)
	assert.Equal(t, wanted.String(), got.Hash)
	assert.Equal(t, "0.1.0", got.Manifest.Package.Version,
		"the worktree must hold the revision that was asked for")
}

func TestResolve_HonoursAnExplicitCommit(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	wanted := f.release(t, "0.1.0")
	f.tag(t, "v0.2.0", f.release(t, "0.2.0"))

	got, err := resolve.Resolve(resolve.Request{URL: testURL, Revision: wanted.String()}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, wanted.String(), got.Hash)
}

func TestResolve_FallsBackToHeadWithoutReleaseTags(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	f.release(t, "0.1.0")
	head := f.release(t, "0.2.0")

	got, err := resolve.Resolve(resolve.Request{URL: testURL}, discardLogger())
	require.NoError(t, err)

	assert.Equal(t, "HEAD", got.Revision)
	assert.Equal(t, head.String(), got.Hash, "an untagged package is still pinned to an exact commit")
}

func TestResolve_RefTakesTheVersionFromTheManifest(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	// The author tagged a release that disagrees with the manifest. Typst
	// requires the install path to match the manifest, so the manifest wins.
	f.tag(t, "v9.9.9", f.release(t, "0.2.0"))

	got, err := resolve.Resolve(resolve.Request{URL: testURL}, discardLogger())
	require.NoError(t, err)

	ref, err := got.Ref("gotpm")
	require.NoError(t, err)
	assert.Equal(t, "@gotpm/my-pkg:0.2.0", ref.String())
	assert.Equal(t, "v9.9.9", got.Revision, "the revision still records what was checked out")
}

func TestResolve_RepositoryWithoutAManifestIsNotAPackage(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	require.NoError(t, paths.WriteFile(filepath.Join(f.dir, "README.md"), []byte("hi")))
	wt, err := f.repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))
	_, err = wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	_, err = resolve.Resolve(resolve.Request{URL: testURL}, discardLogger())
	require.ErrorIs(t, err, resolve.ErrNotAPackage)
}

func TestResolve_RejectsAnUnknownRevision(t *testing.T) { //nolint: paralleltest
	f := seedCachedRepo(t)
	f.tag(t, "v0.1.0", f.release(t, "0.1.0"))

	_, err := resolve.Resolve(resolve.Request{URL: testURL, Revision: "v9.9.9"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "v9.9.9")
}

func TestResolve_ClonesALocalRepository(t *testing.T) { //nolint: paralleltest
	// The only test that exercises the clone itself rather than a cache that
	// was already populated, so the origin lives outside the cache.
	isolateDataDir(t)
	f := newRepo(t, filepath.Join(t.TempDir(), "my-pkg"))
	f.tag(t, "v0.1.0", f.release(t, "0.1.0"))
	wanted := f.release(t, "0.2.0")
	f.tag(t, "v0.2.0", wanted)

	origin := "file://" + f.dir
	got, err := resolve.Resolve(resolve.Request{URL: origin}, discardLogger())
	require.NoError(t, err)

	assert.Equal(t, "v0.2.0", got.Revision)
	assert.Equal(t, wanted.String(), got.Hash)
	assert.Equal(t, "0.2.0", got.Manifest.Package.Version)
	assert.NotEqual(t, f.dir, got.Dir, "the clone must live in the cache, not in the origin")
	assert.FileExists(t, filepath.Join(got.Dir, manifest.FileName))
}

func TestLatestStableTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags []string
		want string
		ok   bool
	}{
		{"no tags", nil, "", false},
		{"nothing that parses", []string{"nightly", "latest", "release"}, "", false},
		{"picks the highest", []string{"v0.1.0", "v0.10.0", "v0.9.0"}, "v0.10.0", true},
		{"order does not matter", []string{"v0.10.0", "v0.1.0"}, "v0.10.0", true},
		{"prereleases are skipped", []string{"v0.2.0", "v0.3.0-rc1"}, "v0.2.0", true},
		{"tags without a v prefix", []string{"0.1.0", "0.2.0"}, "0.2.0", true},
		{"mixed prefixes", []string{"0.3.0", "v0.2.0"}, "0.3.0", true},
		{"major beats minor", []string{"v1.0.0", "v0.99.0"}, "v1.0.0", true},
		{"ignores unrelated tags", []string{"v1.0.0", "release-candidate", "v1.0.0-beta"}, "v1.0.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := resolve.LatestStableTag(tt.tags)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
