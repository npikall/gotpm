package fonts

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
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/npikall/gotpm/internal/lockfile"
)

const (
	// GoogleSource names the Google Fonts repository in a font pin.
	GoogleSource = "google-fonts"
	// GoogleRawURL serves the files of the Google Fonts repository by commit.
	GoogleRawURL = "https://raw.githubusercontent.com/google/fonts"
	// GoogleAPIURL is the GitHub API of the Google Fonts repository.
	GoogleAPIURL = "https://api.github.com/repos/google/fonts"
	// googleRepo is how a pin's URL names the repository, without a scheme.
	googleRepo = "github.com/google/fonts"
	// Timeout bounds every request. Font files run to several megabytes.
	Timeout time.Duration = 60 * time.Second
	// tokenEnvVar lifts the GitHub API rate limit when set.
	tokenEnvVar  = "GITHUB_TOKEN" //nolint: gosec // the variable's name, not a credential
	metadataFile = "METADATA.pb"
)

// licenses are the top-level directories of the Google Fonts repository, each
// holding the families published under that license.
var licenses = []string{"ofl", "apache", "ufl"}

// filenamePattern matches a font file entry of a METADATA.pb, which is a
// protobuf in text format: `filename: "Roboto[wdth,wght].ttf"`.
var filenamePattern = regexp.MustCompile(`(?m)^\s*filename:\s*"([^"]+)"`)

var errNotFound = errors.New("not found")

// Google is the Google Fonts repository on GitHub.
type Google struct {
	RawURL string
	APIURL string
}

// NewGoogle returns the real Google Fonts repository.
func NewGoogle() Google {
	return Google{RawURL: GoogleRawURL, APIURL: GoogleAPIURL}
}

func (Google) Name() string { return GoogleSource }

// Resolve finds the license directory holding the family, pins the newest
// commit that touched it, and lists the font files that commit's METADATA.pb
// names.
func (g Google) Resolve(ctx context.Context, family string) (lockfile.Font, error) {
	key, err := validKey(family)
	if err != nil {
		return lockfile.Font{}, err
	}
	license, err := g.license(ctx, key)
	if err != nil {
		return lockfile.Font{}, err
	}
	dir := license + "/" + key
	hash, files, err := g.newest(ctx, dir)
	if err != nil {
		return lockfile.Font{}, fmt.Errorf("%q: %w", family, err)
	}
	return lockfile.Font{Family: family, Source: GoogleSource, URL: googleRepo + "//" + dir, Hash: hash, Files: files}, nil
}

// Download fetches one file of a pinned family at its pinned commit.
func (g Google) Download(ctx context.Context, pin lockfile.Font, file string) ([]byte, error) {
	dir, err := googleDir(pin.URL)
	if err != nil {
		return nil, err
	}
	return g.raw(ctx, pin.Hash, dir, file)
}

// Families lists the family directories under every license directory.
func (g Google) Families(ctx context.Context) ([]Listing, error) {
	root, err := g.tree(ctx, "main")
	if err != nil {
		return nil, err
	}
	var out []Listing
	for _, dir := range licenseDirs(root) {
		families, err := g.familiesUnder(ctx, dir)
		if err != nil {
			return nil, err
		}
		out = append(out, families...)
	}
	slices.SortFunc(out, func(a, b Listing) int { return strings.Compare(a.Family, b.Family) })
	return out, nil
}

// newest is the newest commit touching dir, and the font files its
// METADATA.pb names.
func (g Google) newest(ctx context.Context, dir string) (string, []lockfile.FontFile, error) {
	hash, err := g.latestCommit(ctx, dir)
	if err != nil {
		return "", nil, err
	}
	metadata, err := g.raw(ctx, hash, dir, metadataFile)
	if err != nil {
		return "", nil, err
	}
	files, err := fontFiles(metadata)
	return hash, files, err
}

func (g Google) familiesUnder(ctx context.Context, license treeEntry) ([]Listing, error) {
	entries, err := g.tree(ctx, license.SHA)
	if err != nil {
		return nil, err
	}
	var out []Listing
	for _, entry := range entries {
		if entry.Type == "tree" {
			out = append(out, Listing{Family: entry.Path, License: license.Path})
		}
	}
	return out, nil
}

func (g Google) license(ctx context.Context, key string) (string, error) {
	for _, license := range licenses {
		_, err := g.raw(ctx, "main", license+"/"+key, metadataFile)
		if errors.Is(err, errNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		return license, nil
	}
	return "", fmt.Errorf("%w in the google fonts repository: %q", ErrFontNotFound, key)
}

func (g Google) latestCommit(ctx context.Context, dir string) (string, error) {
	var commits []struct {
		SHA string `json:"sha"`
	}
	query := url.Values{"path": {dir}, "per_page": {"1"}}
	if err := g.api(ctx, "/commits?"+query.Encode(), &commits); err != nil {
		return "", err
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("%w: no commit touches %q", ErrFontNotFound, dir)
	}
	return commits[0].SHA, nil
}

func (g Google) tree(ctx context.Context, sha string) ([]treeEntry, error) {
	var tree struct {
		Tree []treeEntry `json:"tree"`
	}
	if err := g.api(ctx, "/git/trees/"+url.PathEscape(sha), &tree); err != nil {
		return nil, err
	}
	return tree.Tree, nil
}

// api decodes a GitHub API response. $GITHUB_TOKEN is sent only here, to the
// API it belongs to, never to the host serving raw files.
func (g Google) api(ctx context.Context, path string, target any) error {
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

func (g Google) raw(ctx context.Context, ref, dir, file string) ([]byte, error) {
	requestURL := strings.Join([]string{strings.TrimSuffix(g.RawURL, "/"), url.PathEscape(ref), dir, url.PathEscape(file)}, "/")
	return get(ctx, requestURL, nil)
}

// licenseDirs keeps the license directories of the repository root.
func licenseDirs(root []treeEntry) []treeEntry {
	var dirs []treeEntry
	for _, entry := range root {
		if entry.Type == "tree" && slices.Contains(licenses, entry.Path) {
			dirs = append(dirs, entry)
		}
	}
	return dirs
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

// googleDir is the directory a pin's URL names inside the repository, e.g.
// "ofl/opensans". A pin is read from a lock on disk, so anything else is
// refused rather than requested.
func googleDir(pinURL string) (string, error) {
	dir := strings.TrimPrefix(pinURL, googleRepo+"//")
	license, key, _ := strings.Cut(dir, "/")
	if !slices.Contains(licenses, license) || Key(key) != key || key == "" {
		return "", fmt.Errorf("%w: %q is not a family of %s", ErrInvalidFileRef, pinURL, googleRepo)
	}
	return dir, nil
}

// fontFiles lists the font files a METADATA.pb names, sorted. Each must be a
// plain file name, since it becomes a path inside the family's directory.
func fontFiles(metadata []byte) ([]lockfile.FontFile, error) {
	matches := filenamePattern.FindAllSubmatch(metadata, -1)
	if len(matches) == 0 {
		return nil, ErrNoFontFiles
	}
	files := make([]lockfile.FontFile, 0, len(matches))
	for _, match := range matches {
		name := string(match[1])
		if err := validFileName(name); err != nil {
			return nil, err
		}
		files = append(files, lockfile.FontFile{Name: name})
	}
	slices.SortFunc(files, func(a, b lockfile.FontFile) int { return strings.Compare(a.Name, b.Name) })
	return slices.CompactFunc(files, func(a, b lockfile.FontFile) bool { return a.Name == b.Name }), nil
}

// validFileName accepts a plain file name only, as it becomes a path inside
// the family's directory: no separator, and not "." or "..".
func validFileName(name string) error {
	if name != filepath.Base(name) || strings.ContainsAny(name, `/\`) || strings.Trim(name, ".") == "" {
		return fmt.Errorf("%w: %q", ErrInvalidFileRef, name)
	}
	return nil
}

func get(ctx context.Context, requestURL string, header http.Header) ([]byte, error) {
	req, err := newRequest(ctx, requestURL, header)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: Timeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close() //nolint: errcheck
	return readBody(resp, requestURL)
}

func newRequest(ctx context.Context, requestURL string, header http.Header) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}
	maps.Copy(req.Header, header)
	return req, nil
}

func readBody(resp *http.Response, requestURL string) ([]byte, error) {
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
