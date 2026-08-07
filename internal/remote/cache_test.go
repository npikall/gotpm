package remote_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/remote"
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
