package remote_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateDataDir points gotpm's data directory at a temporary one.
func isolateDataDir(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("HOME", data)
	t.Setenv("APPDATA", data)

	cacheDir, err := remote.CacheDir()
	require.NoError(t, err)
	return cacheDir
}

func TestCachePath_MirrorsTheCanonicalURL(t *testing.T) { //nolint: paralleltest
	cacheDir := isolateDataDir(t)
	tests := []struct {
		name      string
		canonical string
		want      []string
	}{
		{"host owner repo", "github.com/a/cetz", []string{"github.com", "a", "cetz"}},
		{"gitlab subgroup", "gitlab.com/g/sub/cetz", []string{"gitlab.com", "g", "sub", "cetz"}},
		{"local repository", "file:///srv/repos/cetz", []string{"local", "srv", "repos", "cetz"}},
		{"port in host", "localhost:3000/a/cetz", []string{"localhost_3000", "a", "cetz"}},
	}
	// Subtests cannot run in parallel: the data directory is set per test.
	for _, tt := range tests { //nolint: paralleltest
		t.Run(tt.name, func(t *testing.T) {
			got, err := remote.CachePath(tt.canonical)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(append([]string{cacheDir}, tt.want...)...), got)
		})
	}
}

func TestCachePath_SeparatesRepositoriesThatShareAName(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)

	// The whole point of the layout: a flat cache keyed on the repository name
	// would hand one project the other project's code.
	first, err := remote.CachePath("github.com/a/cetz")
	require.NoError(t, err)
	second, err := remote.CachePath("gitlab.com/b/cetz")
	require.NoError(t, err)
	sameHost, err := remote.CachePath("github.com/b/cetz")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "same name on different hosts")
	assert.NotEqual(t, first, sameHost, "same name under different owners")
}

func TestCachePath_StaysInsideTheCacheDirectory(t *testing.T) { //nolint: paralleltest
	cacheDir := isolateDataDir(t)

	for _, canonical := range []string{
		"github.com/../../etc/passwd",
		"github.com/a/../../../..",
		"file:///../../etc",
	} {
		got, err := remote.CachePath(canonical)
		require.NoError(t, err, canonical)
		assert.True(t, strings.HasPrefix(got, cacheDir+string(os.PathSeparator)),
			"%q escaped the cache directory: %q", canonical, got)
	}
}

func TestCachePath_RejectsAnEmptyURL(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)

	_, err := remote.CachePath("")
	require.ErrorIs(t, err, remote.ErrInvalidCacheKey)
}

func TestClearCache_RemovesTheWholeTree(t *testing.T) { //nolint: paralleltest
	cacheDir := isolateDataDir(t)
	dir, err := remote.CachePath("github.com/a/cetz")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o750))

	require.NoError(t, remote.ClearCache())
	assert.NoDirExists(t, cacheDir, "nested clones must go with the cache, not survive it")
}

func TestEnsureClone_ClonesThenReusesTheCache(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	pkg := testrepo.New(t, "cetz", "0.4.0").Release()

	first, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Repo.Close() })
	assert.True(t, first.Cloned)
	_, err = remote.ResolveHash(first.Repo, pkg.Tag())
	require.NoError(t, err, "tags come along with the clone")

	pkg.Release()
	second, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Repo.Close() })
	assert.False(t, second.Cloned, "an existing clone is reused")
	assert.Equal(t, first.Dir, second.Dir)
	_, err = second.Repo.CommitObject(plumbing.NewHash(pkg.Hash()))
	require.NoError(t, err, "a reused clone is fetched first")
}

func TestEnsureClone_RejectsAnEmptyURL(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)

	_, err := remote.EnsureClone("", "", false)
	require.ErrorIs(t, err, remote.ErrInvalidCacheKey)
}

func TestEnsureClone_ReportsABrokenCacheEntry(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	dir, err := remote.CachePath("github.com/a/cetz")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o750))

	_, err = remote.EnsureClone("github.com/a/cetz", "https://github.com/a/cetz", false)
	require.ErrorContains(t, err, "opening cached clone")
}

func TestEnsureClone_ReportsAFailedClone(t *testing.T) { //nolint: paralleltest
	isolateDataDir(t)
	missing := "file://" + filepath.Join(t.TempDir(), "missing")

	_, err := remote.EnsureClone(missing, missing, false)
	require.Error(t, err)
}

func TestEnsureClone_ReachesAPrivateRepositoryWithTheUsersCredentials(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.GrantAccess(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()

	clone, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)

	require.NoError(t, err)
	t.Cleanup(func() { _ = clone.Repo.Close() })
	assert.True(t, clone.Cloned)
	assert.True(t, clone.Private, "it took credentials to reach")
}

func TestEnsureClone_FetchesACachedPrivateRepositoryWithTheUsersCredentials(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.GrantAccess(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()
	first, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)
	require.NoError(t, err)
	require.NoError(t, first.Repo.Close())
	pkg.Release()

	second, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)

	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Repo.Close() })
	assert.False(t, second.Cloned)
	assert.True(t, second.Private, "a cached clone is still known to be private")
	_, err = second.Repo.CommitObject(plumbing.NewHash(pkg.Hash()))
	require.NoError(t, err, "the cached clone was fetched with the credentials")
}

func TestEnsureClone_PrivateRepositoryWithoutCredentialsIsNotFound(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()

	_, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)

	require.ErrorIs(t, err, remote.ErrRepositoryNotFound)
}

func TestEnsureClone_AnonymousLeavesTheUsersCredentialsAlone(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.GrantAccess(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()

	_, err := remote.EnsureClone(pkg.URL(), pkg.URL(), true)

	require.ErrorIs(t, err, remote.ErrRepositoryNotFound)
}

func TestEnsureClone_AnonymousUsesACachedPrivateCloneAsItIs(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.GrantAccess(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()
	first, err := remote.EnsureClone(pkg.URL(), pkg.URL(), false)
	require.NoError(t, err)
	require.NoError(t, first.Repo.Close())

	second, err := remote.EnsureClone(pkg.URL(), pkg.URL(), true)

	require.NoError(t, err, "a fetch that fails leaves the cached clone usable")
	t.Cleanup(func() { _ = second.Repo.Close() })
	assert.False(t, second.Private, "without credentials, nothing says it is private")
}

func TestRefusesAnonymous_APrivateRepositoryEvenWhenTheUserHasCredentials(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.GrantAccess(t)
	pkg := testrepo.New(t, "secret", "0.1.0").Private().Release()

	assert.True(t, remote.RefusesAnonymous(pkg.URL()))
}

func TestRefusesAnonymous_NotAPublicRepository(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	pkg := testrepo.New(t, "open", "0.1.0").Release()

	assert.False(t, remote.RefusesAnonymous(pkg.URL()))
}

func TestCarriesOwnCredentials(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		want bool
	}{
		{"git@github.com:a/cetz.git", true},
		{"github.com:a/cetz", true},
		{"ssh://git@github.com/a/cetz", true},
		{"https://github.com/a/cetz", false},
		{"http://example.com/a/cetz", false},
		{"file:///tmp/a/cetz", false},
		{"/tmp/a/cetz", false},
		{`C:\a\cetz`, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, remote.CarriesOwnCredentials(c.url), c.url)
	}
}
