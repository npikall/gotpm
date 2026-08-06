// Package paths resolves the platform specific directories gotpm reads and
// writes, and holds the small filesystem helpers that go with them.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	ErrDataDirNotResolvable = errors.New("could not resolve typst local package directory")

	ErrNotAFile      = errors.New("path is a directory, not a file")
	ErrNotADirectory = errors.New("path is a file, not a directory")
)

// DataDir returns the platform specific path to the users data directory
func DataDir() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return LinuxDataDir()
	case "darwin":
		return DarwinDataDir()
	case "windows":
		return WindowsDataDir()
	default:
		return "", fmt.Errorf("%w: unsupported OS %q", ErrDataDirNotResolvable, runtime.GOOS)
	}
}

// ConfigDir returns the platform specific path to the users config directory
func ConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get configdir: %w", err)
	}
	return configDir, nil
}

// AppDataDir returns the platform specific path to the apps data directory
func AppDataDir(name string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, name), nil
}

// AppConfigDir returns the platform specific path to the apps config directory
func AppConfigDir(name string) (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, name), nil
}

func FileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("checking file %q: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("checking file %q: %w", path, ErrNotAFile)
	}
	return nil
}

func DirectoryExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("checking directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("checking directory %q: %w", path, ErrNotADirectory)
	}
	return nil
}
