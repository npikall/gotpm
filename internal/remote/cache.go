package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/npikall/gotpm/internal/paths"
)

const (
	cacheDirName  = "remotes"
	localCacheDir = "local"
	fileScheme    = "file://"
)

var ErrInvalidCacheKey = errors.New("cannot derive a cache location")

// CacheDir returns the directory holding cloned remote repositories, without
// creating it.
func CacheDir() (string, error) {
	base, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, cacheDirName), nil
}

// CachePath returns the directory a repository is cached in. The location mirrors
// the canonical url — host, owner, then repository — so two repositories sharing
// a name do not end up in the same clone.
func CachePath(canonicalURL string) (string, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", err
	}
	segments, err := Segments(canonicalURL)
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{cacheDir}, segments...)...), nil
}

// Segments turns a canonical url into the path segments a clone is laid out under
// — host, owner, then repository. Callers supply the directory those segments
// hang off, which keeps clones of different purposes apart.
func Segments(canonicalURL string) ([]string, error) {
	segments := cacheSegments(canonicalURL)
	if len(segments) == 0 {
		return nil, fmt.Errorf("%w for %q", ErrInvalidCacheKey, canonicalURL)
	}
	return segments, nil
}

func cacheSegments(canonicalURL string) []string {
	trimmed := strings.TrimSpace(canonicalURL)

	var prefix []string
	if after, found := strings.CutPrefix(trimmed, fileScheme); found {
		prefix, trimmed = []string{localCacheDir}, after
	}

	segments := prefix
	for raw := range strings.SplitSeq(trimmed, "/") {
		if segment := sanitizeSegment(raw); segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func sanitizeSegment(raw string) string {
	if raw == "" || raw == "." || raw == ".." {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if isSafeRune(r) {
			return r
		}
		return '_'
	}, raw)
}

const safePunctuation = ".-_"

func isSafeRune(r rune) bool {
	return r <= unicode.MaxASCII &&
		(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(safePunctuation, r))
}

// Clone is a repository present in the cache.
type Clone struct {
	Dir  string
	Repo *git.Repository
	// Cloned reports whether the repository was fetched from scratch rather
	// than found in the cache.
	Cloned bool
	// Private reports whether reaching the repository took the user's
	// credentials. A cached clone that could not be fetched at all is not.
	Private bool
}

// EnsureClone returns the cached clone of a repository, cloning it from cloneURL
// when it is not there yet and fetching updates when it is. Nothing is checked
// out. A repository that cannot be reached without credentials is reached with
// the user's own, unless anonymous is set. The caller owns the returned
// repository and must close it.
func EnsureClone(canonicalURL, cloneURL string, anonymous bool) (*Clone, error) {
	repoDir, err := CachePath(canonicalURL)
	if err != nil {
		return nil, err
	}

	if paths.IsDir(repoDir) {
		repo, err := git.PlainOpen(repoDir)
		if err != nil {
			return nil, fmt.Errorf("opening cached clone %q: %w", repoDir, err)
		}
		private := fetchBestEffort(repo, cloneURL, anonymous)
		return &Clone{Dir: repoDir, Repo: repo, Private: private}, nil
	}

	repo, err := CloneWithoutCheckout(cloneURL, repoDir)
	if err == nil {
		return &Clone{Dir: repoDir, Repo: repo, Cloned: true}, nil
	}
	if anonymous || !errors.Is(err, ErrRepositoryNotFound) {
		return nil, err
	}
	auth, ok := credentialsFor(cloneURL)
	if !ok {
		return nil, err
	}
	if err := os.RemoveAll(repoDir); err != nil {
		return nil, fmt.Errorf("clearing failed clone %q: %w", repoDir, err)
	}
	repo, err = CloneWithoutCheckout(cloneURL, repoDir, client.WithHTTPAuth(auth))
	if err != nil {
		return nil, err
	}
	return &Clone{Dir: repoDir, Repo: repo, Cloned: true, Private: true}, nil
}

// fetchBestEffort updates a cached clone, with the user's credentials when it
// cannot be reached without them, and reports whether that was needed. A
// failure is ignored: the cached clone is used as it is.
func fetchBestEffort(repo *git.Repository, cloneURL string, anonymous bool) bool {
	err := repo.Fetch(&git.FetchOptions{Tags: git.AllTags})
	if anonymous || !needsCredentials(err) {
		return false
	}
	auth, ok := credentialsFor(cloneURL)
	if !ok {
		return false
	}
	err = repo.Fetch(&git.FetchOptions{Tags: git.AllTags, ClientOptions: []client.Option{client.WithHTTPAuth(auth)}})
	return err == nil || errors.Is(err, git.NoErrAlreadyUpToDate)
}

// ClearCache removes every cloned remote repository.
func ClearCache() error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("could not remove remotes cache %q: %w", dir, err)
	}
	return nil
}
