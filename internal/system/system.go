package system

import (
	"errors"
	"os"
	"path/filepath"

	i "github.com/npikall/gotpm/internal/manifest"
)

const (
	MACOS   string = "darwin"
	WINDOWS string = "windows"
	LINUX   string = "linux"
)

var ErrOperatingSystem = errors.New("unsupported operating system")

// Get the Path to the systems data directory
func GetDataDirectory(os string, home string) (string, error) {
	switch os {
	case MACOS:
		return filepath.Join(home, "Library/Application Support"), nil
	case WINDOWS:
		return getWindowsPath(home), nil
	case LINUX:
		return getLinuxPath(home), nil
	default:
		return "", ErrOperatingSystem
	}
}

func getWindowsPath(home string) string {
	env := os.Getenv("APPDATA")
	if env == "" {
		return filepath.Join(home, "AppData/Roaming")
	}
	return env
}
func getLinuxPath(home string) string {
	env := os.Getenv("XDG_DATA_HOME")
	if env == "" {
		return filepath.Join(home, ".local/share")
	}
	return env
}

// Get the final path to a package in the data directory given a namespace and a name.
func GetStoragePath(goos, home, namespace, name, version string) (string, error) {
	// TODO: Add test for this
	dataDir, err := GetDataDirectory(goos, home)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "typst", "packages", namespace, name, version)
	return path, nil
}

// Get the path to $DATA_DIR/typst/packages
func GetTypstPath(goos, home string) (string, error) {
	dataDir, err := GetDataDirectory(goos, home)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "typst", "packages")
	return path, nil
}

// Try to open the Typst TOML file. Returns an error if not existing.
func OpenTypstTOML(directory string) (i.PackageInfo, error) {
	tomlPath := filepath.Join(directory, "typst.toml")
	_, err := os.Stat(tomlPath)
	if err != nil {
		return i.PackageInfo{}, err
	}

	tomlContent, err := os.ReadFile(tomlPath)
	if err != nil {
		return i.PackageInfo{}, err
	}

	return i.TypstTOMLUnmarshal(tomlContent)
}
