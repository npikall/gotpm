package internal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/npikall/gotpm/internal"
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

// TestIndexCache_IsValid locks the 1-hour TTL boundary.
func TestIndexCache_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		timestamp time.Time
		want      bool
	}{
		{"fresh (now)", time.Now(), true},
		{"just under TTL", time.Now().Add(-(CacheTTL - time.Second)), true},
		{"exactly TTL", time.Now().Add(-CacheTTL), false},
		{"over TTL", time.Now().Add(-(CacheTTL + time.Second)), false},
		{"zero time", time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &IndexCache{Timestamp: tt.timestamp}
			assert.Equal(t, tt.want, c.IsValid())
		})
	}
}

// TestResolveCachePath locks the path structure: ends with gotpm/index-cache.json.
func TestResolveCachePath(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := ResolveCachePath()
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, filepath.Join("gotpm", "index-cache.json")),
		"path %q must end with gotpm/index-cache.json", path)
}

// TestLoadIndexCache_Missing locks nil,nil when file does not exist.
func TestLoadIndexCache_Missing(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	cache, err := LoadIndexCache()
	require.Error(t, err)
	assert.Nil(t, cache)
}

// TestLoadIndexCache_InvalidJSON locks error on malformed file.
func TestLoadIndexCache_InvalidJSON(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := ResolveCachePath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte("not json{{{"), 0o600))

	cache, err := LoadIndexCache()
	require.Error(t, err)
	assert.Nil(t, cache)
}

// TestLoadIndexCache_Valid locks correct parse of a well-formed cache file.
func TestLoadIndexCache_Valid(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	path, err := ResolveCachePath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))

	timestamp := time.Now().Truncate(time.Second)
	fixture := IndexCache{
		Timestamp: timestamp,
		Index:     map[string]string{"pkg-a": "0.1.0", "pkg-b": "2.3.4"},
	}
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	got, err := LoadIndexCache()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, timestamp.UTC(), got.Timestamp.UTC())
	assert.Equal(t, fixture.Index, got.Index)
}

// TestSaveIndexCache locks file creation, dir creation, and written content.
func TestSaveIndexCache(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	index := map[string]string{"foo": "1.0.0", "bar": "0.2.1"}

	before := time.Now().Truncate(time.Second)
	require.NoError(t, SaveIndexCache(index))
	after := time.Now().Add(time.Second)

	path, err := ResolveCachePath()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got IndexCache
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, index, got.Index)
	assert.False(t, got.Timestamp.Before(before), "timestamp must not be before save start")
	assert.False(t, got.Timestamp.After(after), "timestamp must not be after save end")
}

// TestSaveLoadRoundtrip locks that Save then Load returns identical data.
func TestSaveLoadRoundtrip(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	index := map[string]string{"typst-pkg": "3.1.4", "other": "0.0.1"}

	require.NoError(t, SaveIndexCache(index))

	got, err := LoadIndexCache()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, index, got.Index)
	assert.True(t, got.IsValid(), "freshly saved cache must be valid")
}

// TestSaveIndexCache_CreatesDir locks that SaveIndexCache creates missing parent dirs.
func TestSaveIndexCache_CreatesDir(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	// Verify parent dir does not exist before save.
	path, err := ResolveCachePath()
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Dir(path))
	assert.True(t, os.IsNotExist(statErr), "dir must not exist before first save")

	require.NoError(t, SaveIndexCache(map[string]string{}))

	_, err = os.Stat(path)
	assert.NoError(t, err, "cache file must exist after save")
}
