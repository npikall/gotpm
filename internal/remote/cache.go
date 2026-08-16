package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/paths"
)

const (
	cacheDirName = "remotes"
	// localCacheDir holds clones of repositories that live on this machine
	// rather than on a host, keeping them apart from hosted ones.
	localCacheDir = "local"
	// fileScheme marks such a repository.
	fileScheme = "file://"
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

// CachePath returns the directory a repository is cached in.
//
// The location mirrors the canonical url — host, owner, then repository — so
// two repositories that happen to share a name, whether on different hosts or
// under different owners, do not end up in the same clone.
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

// Segments turns a canonical url into the path segments a clone of it is laid
// out under — host, owner, then repository. Callers supply the directory those
// segments hang off, which is what keeps clones of different purposes apart
// while they agree on where one url belongs.
func Segments(canonicalURL string) ([]string, error) {
	segments := cacheSegments(canonicalURL)
	if len(segments) == 0 {
		return nil, fmt.Errorf("%w for %q", ErrInvalidCacheKey, canonicalURL)
	}
	return segments, nil
}

// cacheSegments turns a canonical url into the path segments it is cached
// under, dropping anything that cannot safely name a directory.
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

// sanitizeSegment reduces one path segment to characters that name a directory
// on every platform gotpm runs on, and refuses the ones that would escape the
// cache directory.
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

// safePunctuation is what may appear in a directory name besides letters and
// digits. Hosts and repository names are full of dots and dashes.
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
}

// EnsureClone returns the cached clone of a repository, cloning it from
// cloneURL when it is not there yet and fetching updates when it is. Nothing is
// checked out: which revision a caller wants may depend on the tags the fetch
// has just brought in.
//
// The cache is keyed on canonicalURL. The caller owns the returned repository
// and must close it.
func EnsureClone(canonicalURL, cloneURL string) (*Clone, error) {
	repoDir, err := CachePath(canonicalURL)
	if err != nil {
		return nil, err
	}

	if paths.IsDir(repoDir) {
		repo, err := git.PlainOpen(repoDir)
		if err != nil {
			return nil, fmt.Errorf("opening cached clone %q: %w", repoDir, err)
		}
		// A failed fetch is not fatal: the cached clone may already hold the
		// revision that was asked for, and that keeps gotpm usable offline.
		_ = repo.Fetch(&git.FetchOptions{Tags: git.AllTags})
		return &Clone{Dir: repoDir, Repo: repo}, nil
	}

	repo, err := CloneWithoutCheckout(cloneURL, repoDir)
	if err != nil {
		return nil, err
	}
	return &Clone{Dir: repoDir, Repo: repo, Cloned: true}, nil
}

// Ensure returns the local clone of a remote repository, checked out at rev,
// cloning it into the cache when it is not there yet. It reports whether a
// clone was performed.
func Ensure(remoteURL, rev string) (string, bool, error) {
	clone, err := EnsureClone(remoteURL, DefaultHTTPCloneURL(remoteURL))
	if err != nil {
		return "", false, err
	}
	defer clone.Repo.Close() //nolint: errcheck

	if rev != "" {
		if err := CheckoutRevision(clone.Repo, rev); err != nil {
			return "", false, err
		}
	}
	return clone.Dir, clone.Cloned, nil
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
