package remote

import (
	"fmt"
	"os"
	"path/filepath"

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
