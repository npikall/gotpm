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

// Ensure returns the local clone of a remote repository, checked out at rev,
// cloning it into the cache when it is not there yet. It reports whether a
// clone was performed.
func Ensure(remoteURL, rev string) (string, bool, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", false, err
	}
	name, err := RepoNameFromURL(remoteURL)
	if err != nil {
		return "", false, err
	}
	repoDir := filepath.Join(cacheDir, name)

	if paths.IsDir(repoDir) {
		if rev == "" {
			return repoDir, false, nil
		}
		repo, err := git.PlainOpen(repoDir)
		if err != nil {
			return "", false, fmt.Errorf("opening cached clone %q: %w", repoDir, err)
		}
		_ = repo.Fetch(&git.FetchOptions{})
		if err := CheckoutRevision(repo, rev); err != nil {
			return "", false, err
		}
		return repoDir, false, nil
	}

	if err := CloneRepo(DefaultHTTPCloneURL(remoteURL), repoDir, rev); err != nil {
		return "", false, err
	}
	return repoDir, true, nil
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
