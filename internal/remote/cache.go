package remote

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/paths"
)

const cacheDirName = "remotes"

// CacheDir returns the directory holding cloned remote repositories, without
// creating it.
func CacheDir() (string, error) {
	base, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, cacheDirName), nil
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
	cacheDir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	name, err := RepoNameFromURL(canonicalURL)
	if err != nil {
		return nil, err
	}
	repoDir := filepath.Join(cacheDir, name)

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
