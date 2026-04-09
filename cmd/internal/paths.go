package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	typstPackagesRelPath = "typst/packages"
	DefaultNamespace     = "local"
)

var ErrDataDirNotResolvable = errors.New("could not resolve typst local package directory")

// ResolveLocalPackageDirPath returns the path to the typst packages directory
// without creating it. Respects the $TYPST_PACKAGE_PATH override.
func ResolveLocalPackageDirPath() (string, error) {
	if override := os.Getenv("TYPST_PACKAGE_PATH"); override != "" {
		return override, nil
	}
	base, err := resolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, typstPackagesRelPath), nil
}

// ResolveLocalPackageDir returns the path to the typst packages directory,
// creating it if it does not exist.
func ResolveLocalPackageDir() (string, error) {
	localPkgDir, err := ResolveLocalPackageDirPath()
	if err != nil {
		return "", err
	}
	if err := EnsureDir(localPkgDir); err != nil {
		return "", err
	}
	return localPkgDir, nil
}

// EnsureDir creates dir and all necessary parents.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("creating directory %q: %w", path, err)
	}
	return nil
}

func resolveDataDir() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return ResolveLinuxDataDir()
	case "darwin":
		return ResolveDarwinDataDir()
	case "windows":
		return ResolveWindowsDataDir()
	default:
		return "", fmt.Errorf("%w: unsupported OS %q", ErrDataDirNotResolvable, runtime.GOOS)
	}
}

func ResolveLinuxDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, ".local", "share"), nil
}

func ResolveDarwinDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}

func ResolveWindowsDataDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("%w: %%APPDATA%% is not set", ErrDataDirNotResolvable)
	}
	return appData, nil
}
