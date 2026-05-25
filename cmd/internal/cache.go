package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	CacheTTL      = time.Hour
	cacheAppDir   = "gotpm"
	cacheFileName = "index-cache.json"
)

type IndexCache struct {
	Timestamp time.Time         `json:"timestamp"`
	Index     map[string]string `json:"index"`
}

func (c *IndexCache) IsValid() bool {
	return time.Since(c.Timestamp) < CacheTTL
}

func ResolveCachePath() (string, error) {
	base, err := resolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, cacheAppDir, cacheFileName), nil
}

// LoadIndexCache reads the on-disk cache. Returns nil, nil when the file does not exist.
func LoadIndexCache() (*IndexCache, error) {
	path, err := ResolveCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cache does not exist: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not open cache: %w", err)
	}
	var cache IndexCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("could not unmarshal cache: %w", err)
	}
	return &cache, nil
}

// SaveIndexCache writes the index to disk with the current timestamp.
func SaveIndexCache(index map[string]string) error {
	path, err := ResolveCachePath()
	if err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	cache := IndexCache{
		Timestamp: time.Now(),
		Index:     index,
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("could not marshal index: %w", err)
	}
	err = os.WriteFile(path, data, 0o600) //nolint: mnd
	if err != nil {
		return fmt.Errorf("could not write cache: %w", err)
	}
	return nil
}
