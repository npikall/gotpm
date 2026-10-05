package paths

import "path/filepath"

// GotpmDataDir returns the platform specific path to gotpm's data directory
func GotpmDataDir() (string, error) {
	return AppDataDir("gotpm")
}

// GotpmConfigDir returns the platform specific path to gotpm's config directory
func GotpmConfigDir() (string, error) {
	return AppConfigDir("gotpm")
}

// GotpmForksDir returns the directory the fork clones are kept under, one
// clone per fork (ADR 0006), without creating it.
func GotpmForksDir() (string, error) {
	dataDir, err := GotpmDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "forks"), nil
}
