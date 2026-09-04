package paths

import (
	"os"
	"path/filepath"
)

const (
	typstPackagesRelPath = "typst/packages"

	DefaultNamespace = "local"

	// InstallDirEnvVar is the environment variable that overrides the package directory.
	InstallDirEnvVar = "GOTPM_INSTALL_DIR"
	// InstallDirFlag is the cobra flag name used to override the package directory.
	InstallDirFlag = "install-dir"
	// TypstPackagePathEnvVar is typst's own override for the package directory.
	TypstPackagePathEnvVar = "TYPST_PACKAGE_PATH"
	// ProvenanceFile is the name of the file recording where an installed package
	// came from. Defined here so pkgfiles can exclude it from a package's contents
	// without store and pkgfiles importing one another.
	ProvenanceFile = ".gotpm.json"
)

// Origin says which of the possible sources decided the package directory.
type Origin int

const (
	// OriginDefault means the OS specific default location is used.
	OriginDefault Origin = iota
	// OriginInstallEnv means $GOTPM_INSTALL_DIR decided the location.
	OriginInstallEnv
	// OriginTypstEnv means $TYPST_PACKAGE_PATH decided the location.
	OriginTypstEnv
)

// EnvVar names the environment variable behind an origin, or "" when the
// location was not overridden.
func (o Origin) EnvVar() string {
	switch o {
	case OriginInstallEnv:
		return InstallDirEnvVar
	case OriginTypstEnv:
		return TypstPackagePathEnvVar
	case OriginDefault:
		return ""
	default:
		return ""
	}
}

// PackagesDir reports where packages land for a plain command run, and why.
// It follows the same precedence as InstallDir, minus the --install-dir flag,
// which is per command rather than ambient.
func PackagesDir() (string, Origin, error) {
	if env := os.Getenv(InstallDirEnvVar); env != "" {
		return env, OriginInstallEnv, nil
	}
	if env := os.Getenv(TypstPackagePathEnvVar); env != "" {
		return env, OriginTypstEnv, nil
	}
	dir, err := TypstPackagesDir()
	return dir, OriginDefault, err
}

// TypstPackagesDir returns the path to the typst packages directory without
// creating it. Respects the $TYPST_PACKAGE_PATH override.
func TypstPackagesDir() (string, error) {
	if override := os.Getenv(TypstPackagePathEnvVar); override != "" {
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

// InstallDir returns the package directory path without creating it, and whether
// an override was given: --install-dir, then $GOTPM_INSTALL_DIR, then the OS
// default. An override is the final destination itself (ADR 0003).
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
