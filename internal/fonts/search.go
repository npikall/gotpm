package fonts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/npikall/gotpm/internal/paths"
)

const (
	// IndexMaxAge is how long a fetched family list is searched before it is
	// fetched again.
	IndexMaxAge = 24 * time.Hour
	indexFile   = "font-index.json"
)

// IndexPath is where the family list is cached, beside the package index.
func IndexPath() (string, error) {
	dataDir, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, indexFile), nil
}

// Index searches the families a source offers through a cached list. The list
// is cache: deleting it loses nothing.
type Index struct {
	Source Source
	Path   string
	// Now is the clock the cache age is measured with; nil means time.Now.
	Now func() time.Time
}

// OpenIndex returns the index of the default source, cached in gotpm's data
// directory.
func OpenIndex() (Index, error) {
	path, err := IndexPath()
	if err != nil {
		return Index{}, err
	}
	return Index{Source: Default(), Path: path}, nil
}

type cachedIndex struct {
	Source   string    `json:"source"`
	Fetched  time.Time `json:"fetched"`
	Families []Listing `json:"families"`
}

// Search lists the families whose key contains the query's, so "Open Sans"
// finds "opensans" and "opensanscondensed". An empty query lists them all.
// refresh fetches the list even when the cached one is fresh.
func (i Index) Search(ctx context.Context, query string, refresh bool) ([]Listing, error) {
	families, err := i.families(ctx, refresh)
	if err != nil {
		return nil, err
	}
	key := Key(query)
	var out []Listing
	for _, listing := range families {
		if strings.Contains(listing.Family, key) {
			out = append(out, listing)
		}
	}
	return out, nil
}

func (i Index) families(ctx context.Context, refresh bool) ([]Listing, error) {
	if !refresh {
		if cached, ok := i.load(); ok {
			return cached.Families, nil
		}
	}
	families, err := i.Source.Families(ctx)
	if err != nil {
		return nil, err
	}
	return families, i.save(cachedIndex{Source: i.Source.Name(), Fetched: i.now(), Families: families})
}

// load returns the cached list when it is from this source and fresh enough.
// Anything unreadable is treated as no cache, since it is refetched anyway.
func (i Index) load() (cachedIndex, bool) {
	data, err := os.ReadFile(i.Path)
	if err != nil {
		return cachedIndex{}, false
	}
	var cached cachedIndex
	if err := json.Unmarshal(data, &cached); err != nil {
		return cachedIndex{}, false
	}
	fresh := i.now().Sub(cached.Fetched) < IndexMaxAge
	return cached, fresh && cached.Source == i.Source.Name()
}

func (i Index) save(cached cachedIndex) error {
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("could not marshal the font index: %w", err)
	}
	if err := paths.EnsureDir(filepath.Dir(i.Path)); err != nil {
		return err
	}
	return paths.WriteFile(i.Path, data)
}

// ClearIndex deletes the cached family list. A missing one is not an error.
func ClearIndex() error {
	path, err := IndexPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not remove %q: %w", path, err)
	}
	return nil
}

func (i Index) now() time.Time {
	if i.Now == nil {
		return time.Now()
	}
	return i.Now()
}
