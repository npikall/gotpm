package index_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redirectCacheToTempDir points all OS data-dir env vars to a temp directory
// so cache operations do not touch the real user data dir.
func redirectCacheToTempDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("APPDATA", filepath.Join(tmp, "appdata"))
}

// TestCache_IsValid locks the 1-hour TTL boundary.
func TestCache_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		timestamp time.Time
		want      bool
	}{
		{"fresh (now)", time.Now(), true},
		{"just under TTL", time.Now().Add(-(index.CacheTTL - time.Second)), true},
		{"exactly TTL", time.Now().Add(-index.CacheTTL), false},
		{"over TTL", time.Now().Add(-(index.CacheTTL + time.Second)), false},
		{"zero time", time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &index.Cache{Timestamp: tt.timestamp}
			assert.Equal(t, tt.want, c.IsValid())
		})
	}
}

// TestCachePath locks the path structure: ends with gotpm/index-cache.json.
func TestCachePath(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := index.CachePath()
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, filepath.Join("gotpm", "index-cache.json")),
		"path %q must end with gotpm/index-cache.json", path)
}

func TestLoadCache_Missing(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	cache, err := index.LoadCache()
	require.Error(t, err)
	assert.Nil(t, cache)
}

func TestLoadCache_InvalidJSON(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := index.CachePath()
	require.NoError(t, err)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))
	require.NoError(t, paths.WriteFile(path, []byte("not json{{{")))

	cache, err := index.LoadCache()
	require.Error(t, err)
	assert.Nil(t, cache)
}

func TestLoadCache_Valid(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := index.CachePath()
	require.NoError(t, err)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))

	timestamp := time.Now().Truncate(time.Second)
	fixture := index.Cache{
		Timestamp: timestamp,
		Index:     index.Index{"pkg-a": "0.1.0", "pkg-b": "2.3.4"},
	}
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, paths.WriteFile(path, data))

	got, err := index.LoadCache()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, timestamp.UTC(), got.Timestamp.UTC())
	assert.Equal(t, fixture.Index, got.Index)
}

func TestSaveCache(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	idx := index.Index{"foo": "1.0.0", "bar": "0.2.1"}

	before := time.Now().Truncate(time.Second)
	require.NoError(t, index.SaveCache(idx))
	after := time.Now().Add(time.Second)

	path, err := index.CachePath()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got index.Cache
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, idx, got.Index)
	assert.False(t, got.Timestamp.Before(before), "timestamp must not be before save start")
	assert.False(t, got.Timestamp.After(after), "timestamp must not be after save end")
}

func TestSaveLoadRoundtrip(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	idx := index.Index{"typst-pkg": "3.1.4", "other": "0.0.1"}

	require.NoError(t, index.SaveCache(idx))

	got, err := index.LoadCache()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, idx, got.Index)
	assert.True(t, got.IsValid(), "freshly saved cache must be valid")
}

func TestSaveCache_CreatesDir(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := index.CachePath()
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Dir(path))
	assert.True(t, os.IsNotExist(statErr), "dir must not exist before first save")

	require.NoError(t, index.SaveCache(index.Index{}))

	_, err = os.Stat(path)
	assert.NoError(t, err, "cache file must exist after save")
}

func TestClearCache(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	require.NoError(t, index.SaveCache(index.Index{"pkg": "1.0.0"}))

	require.NoError(t, index.ClearCache())

	path, err := index.CachePath()
	require.NoError(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "cache file must be gone")

	assert.NoError(t, index.ClearCache(), "clearing a missing cache is not an error")
}
