package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	typstPackagesRelPath = "typst/packages"
	DefaultNamespace     = "local"

	// InstallDirEnvVar is the environment variable that overrides the package directory.
	InstallDirEnvVar = "GOTPM_INSTALL_DIR"
	// InstallDirFlag is the cobra flag name used to override the package directory.
	InstallDirFlag = "install-dir"
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

// ResolvePackageDirPath returns the package directory path without creating it,
// and whether an override was provided (flag or env).
// Resolution order: --install-dir flag > $GOTPM_INSTALL_DIR env > OS default.
// When overridden, the returned path is the final destination — callers must not
// append namespace/name/version sub-directories.
func ResolvePackageDirPath(cmd *cobra.Command) (path string, overridden bool, err error) {
	dir, err := cmd.Flags().GetString(InstallDirFlag)
	if err != nil {
		return "", false, err
	}
	if dir != "" {
		return dir, true, nil
	}
	if env := os.Getenv(InstallDirEnvVar); env != "" {
		return env, true, nil
	}
	p, err := ResolveLocalPackageDirPath()
	return p, false, err
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
	if err := os.MkdirAll(path, 0o755); err != nil {
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
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, ".local", "share"), nil
}

func ResolveDarwinDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
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
