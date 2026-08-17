package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/npikall/gotpm/internal/paths"
)

const (
	// CacheTTL is how long a cached index is used before it is fetched again.
	CacheTTL      = time.Hour
	cacheFileName = "index-cache.json"
)

// Cache is the on-disk form of a fetched index.
type Cache struct {
	Timestamp time.Time `json:"timestamp"`
	Index     Index     `json:"index"`
}

// IsValid reports whether the cache is still within its TTL.
func (c *Cache) IsValid() bool {
	return time.Since(c.Timestamp) < CacheTTL
}

// CachePathIn returns the path of the cache file inside dir.
func CachePathIn(dir string) string {
	return filepath.Join(dir, cacheFileName)
}

// CachePath returns the path of the cache file, without creating it.
func CachePath() (string, error) {
	base, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	return CachePathIn(base), nil
}

// LoadCache reads the cache from disk.
func LoadCache() (*Cache, error) {
	path, err := CachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) //nolint: gosec
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cache does not exist: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not open cache: %w", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("could not unmarshal cache: %w", err)
	}
	return &cache, nil
}

// SaveCacheAt writes the index into dir with the current timestamp. It takes
// the directory rather than reading it from the environment, so a test can
// seed a cache for a gotpm running in another process.
func SaveCacheAt(dir string, idx Index) error {
	path := CachePathIn(dir)
	if err := paths.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.Marshal(Cache{Timestamp: time.Now(), Index: idx})
	if err != nil {
		return fmt.Errorf("could not marshal index: %w", err)
	}
	return paths.WriteFile(path, data)
}

// SaveCache writes the index to disk with the current timestamp.
func SaveCache(idx Index) error {
	base, err := paths.GotpmDataDir()
	if err != nil {
		return err
	}
	return SaveCacheAt(base, idx)
}

// ClearCache removes the cache file. A missing file is not an error.
func ClearCache() error {
	path, err := CachePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove index cache %q: %w", path, err)
	}
	return nil
}
