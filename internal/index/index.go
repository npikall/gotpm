// Package index provides access to the Typst Universe package index: fetching
// it, caching it on disk and looking up the latest version of a package.
package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/npikall/gotpm/internal/semver"
)

const (
	// PackageEndpoint lists the published versions of a single package.
	PackageEndpoint = "https://api.github.com/repos/typst/packages/contents/packages/preview/"
	// URL serves the full package index of the Typst Universe.
	URL = "https://packages.typst.org/preview/index.json"
	// Timeout bounds every request made by this package.
	Timeout time.Duration = 5 * time.Second
)

var ErrHTTPFailedRequest = errors.New("http request failed")

// Index maps a package name to its latest published version.
type Index map[string]string

// Latest returns the latest known version of a package.
func (i Index) Latest(name string) (string, bool) {
	version, ok := i[name]
	return version, ok
}

// Entry is a single record of the published index. The same package appears
// once per released version.
type Entry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Release is one published version directory of a package, as reported by the
// GitHub contents API.
type Release struct {
	Name string `json:"name" validate:"semver"`
}

// Opts controls how Load obtains the index.
type Opts struct {
	// NoCache skips both reading and writing the on-disk cache.
	NoCache bool
}

// Load returns the package index, preferring a still-valid on-disk cache over
// fetching it. A successful fetch refreshes the cache.
func Load(ctx context.Context, opts Opts) (Index, error) {
	if !opts.NoCache {
		if cache, err := LoadCache(); err == nil && cache.IsValid() {
			return cache.Index, nil
		}
	}
	entries, err := Fetch(ctx)
	if err != nil {
		return nil, err
	}
	idx := Build(entries)
	if !opts.NoCache {
		_ = SaveCache(idx)
	}
	return idx, nil
}

// Fetch downloads the full package index of the Typst Universe.
func Fetch(ctx context.Context) ([]Entry, error) {
	return FetchFrom(ctx, URL)
}

// FetchFrom downloads a package index from an arbitrary URL.
func FetchFrom(ctx context.Context, indexURL string) ([]Entry, error) {
	var entries []Entry
	if err := getJSON(ctx, indexURL, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Build reduces a list of index entries to the latest version per package.
func Build(entries []Entry) Index {
	idx := make(Index)
	for _, entry := range entries {
		current, exists := idx[entry.Name]
		if !exists {
			idx[entry.Name] = entry.Version
			continue
		}
		currentV, err := semver.Parse(current)
		if err != nil {
			continue
		}
		entryV, err := semver.Parse(entry.Version)
		if err != nil {
			continue
		}
		if entryV.Compare(currentV) > 0 {
			idx[entry.Name] = entry.Version
		}
	}
	return idx
}

// LatestOnGitHub looks a package up through the GitHub API, for packages the
// index does not know about or when the index is unavailable.
func LatestOnGitHub(ctx context.Context, pkgName string) (string, error) {
	apiURL, err := url.JoinPath(PackageEndpoint, pkgName)
	if err != nil {
		return "", fmt.Errorf("could not create url for %q: %w", pkgName, err)
	}
	releases, err := FetchReleases(ctx, apiURL)
	if err != nil {
		return "", fmt.Errorf("could not find package %q on github: %w", pkgName, err)
	}
	latest, err := LatestRelease(releases)
	if err != nil {
		return "", fmt.Errorf("could not get latest version from response for %q: %w", pkgName, err)
	}
	return latest, nil
}

// FetchReleases lists the published versions of a package.
func FetchReleases(ctx context.Context, releasesURL string) ([]Release, error) {
	var releases []Release
	if err := getJSON(ctx, releasesURL, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// LatestRelease returns the highest version of the given releases.
func LatestRelease(releases []Release) (string, error) {
	candidate := &semver.Version{}
	for _, release := range releases {
		version, err := semver.Parse(release.Name)
		if err != nil {
			return "", err
		}
		if candidate.Compare(version) < 0 {
			candidate = version
		}
	}
	return candidate.String(), nil
}

func getJSON(ctx context.Context, requestURL string, target any) error {
	client := &http.Client{Timeout: Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("could not create new request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close() //nolint: errcheck

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%w with status %s for %s", ErrHTTPFailedRequest, resp.Status, requestURL)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("could not decode response: %w", err)
	}
	return nil
}
