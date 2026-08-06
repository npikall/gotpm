package paths

import (
	"os"
	"path/filepath"
)

const (
	typstPackagesRelPath = "typst/packages"

	// DefaultNamespace is the namespace a package is installed into unless
	// another one is requested.
	DefaultNamespace = "local"

	// InstallDirEnvVar is the environment variable that overrides the package directory.
	InstallDirEnvVar = "GOTPM_INSTALL_DIR"
	// InstallDirFlag is the cobra flag name used to override the package directory.
	InstallDirFlag = "install-dir"
)

// TypstPackagesDir returns the path to the typst packages directory without
// creating it. Respects the $TYPST_PACKAGE_PATH override.
func TypstPackagesDir() (string, error) {
	if override := os.Getenv("TYPST_PACKAGE_PATH"); override != "" {
		return override, nil
	}
	base, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, typstPackagesRelPath), nil
}

// EnsureTypstPackagesDir returns the path to the typst packages directory,
// creating it if it does not exist.
func EnsureTypstPackagesDir() (string, error) {
	dir, err := TypstPackagesDir()
	if err != nil {
		return "", err
	}
	if err := EnsureDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// InstallDir returns the package directory path without creating it, and
// whether an override was provided (flag or env).
// Resolution order: --install-dir flag > $GOTPM_INSTALL_DIR env > OS default.
// When overridden, the returned path is the final destination — callers must not
// append namespace/name/version sub-directories.
func InstallDir(override string) (string, bool, error) {
	if override != "" {
		return override, true, nil
	}
	if env := os.Getenv(InstallDirEnvVar); env != "" {
		return env, true, nil
	}
	dir, err := TypstPackagesDir()
	return dir, false, err
}
