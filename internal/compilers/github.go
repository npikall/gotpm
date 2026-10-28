package compilers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// GitHubAPIURL is the GitHub API of the Typst repository.
	GitHubAPIURL = "https://api.github.com/repos/typst/typst"
	// Timeout bounds every request. A compiler archive runs to tens of
	// megabytes.
	Timeout time.Duration = 5 * time.Minute
	// tokenEnvVar lifts the GitHub API rate limit when set.
	tokenEnvVar = "GITHUB_TOKEN" //nolint: gosec // the variable's name, not a credential
	// perPage is the most releases GitHub lists at once, more than Typst
	// has published.
	perPage = 100
)

var errNotFound = errors.New("not found")

// GitHub reads the releases of the Typst repository on GitHub.
type GitHub struct {
	APIURL string
}

// NewGitHub returns the real Typst repository.
func NewGitHub() GitHub {
	return GitHub{APIURL: GitHubAPIURL}
}

// Releases lists the published releases, drafts left out.
func (g GitHub) Releases(ctx context.Context) ([]Release, error) {
	var listed []ghRelease
	if err := g.api(ctx, "/releases?per_page="+strconv.Itoa(perPage), &listed); err != nil {
		return nil, err
	}
	releases := make([]Release, 0, len(listed))
	for _, r := range listed {
		if !r.Draft {
			releases = append(releases, r.release())
		}
	}
	return releases, nil
}

// Release returns the release tagged "v<version>".
func (g GitHub) Release(ctx context.Context, version string) (Release, error) {
	var r ghRelease
	err := g.api(ctx, "/releases/tags/"+url.PathEscape("v"+version), &r)
	if errors.Is(err, errNotFound) {
		return Release{}, fmt.Errorf("%w: %s", ErrNoRelease, version)
	}
	if err != nil {
		return Release{}, err
	}
	return r.release(), nil
}

// Download fetches an asset from where the release says it is served. The
// token is not sent along: assets are served by another host.
func (GitHub) Download(ctx context.Context, asset Asset) ([]byte, error) {
	return get(ctx, asset.URL, nil)
}

// api decodes a GitHub API response, sending $GITHUB_TOKEN when set.
func (g GitHub) api(ctx context.Context, path string, target any) error {
	header := http.Header{"Accept": {"application/vnd.github+json"}}
	if token := os.Getenv(tokenEnvVar); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	data, err := get(ctx, strings.TrimSuffix(g.APIURL, "/")+path, header)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("could not decode response of %s: %w", path, err)
	}
	return nil
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Prerelease bool      `json:"prerelease"`
	Draft      bool      `json:"draft"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

func (r ghRelease) release() Release {
	assets := make([]Asset, 0, len(r.Assets))
	for _, a := range r.Assets {
		assets = append(assets, Asset(a))
	}
	return Release{Version: strings.TrimPrefix(r.TagName, "v"), Prerelease: r.Prerelease, Assets: assets}
}

func get(ctx context.Context, requestURL string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}
	maps.Copy(req.Header, header)
	resp, err := (&http.Client{Timeout: Timeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close() //nolint: errcheck
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%w with status %s for %s", ErrFailedRequest, resp.Status, requestURL)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", requestURL, err)
	}
	return data, nil
}
