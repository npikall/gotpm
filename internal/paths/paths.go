package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const (
	MACOS   string = "darwin"
	WINDOWS string = "windows"
	LINUX   string = "linux"
)

var ErrOperatingSystem = errors.New("unsupported operating system")

// Get the Path to the systems data directory
func GetPlatformDataDirectory(os string, home string) (string, error) {
	switch os {
	case MACOS:
		return GetMacosDataDir(home), nil
	case WINDOWS:
		return GetWindowsDataDir(home), nil
	case LINUX:
		return GetLinuxDataDir(home), nil
	default:
		return "", ErrOperatingSystem
	}
}

// Get the MacOS Data Directory
func GetMacosDataDir(home string) string {
	return filepath.Join(home, "Library/Application Support")
}

// Get the Windows Data Directory
func GetWindowsDataDir(home string) string {
	env := os.Getenv("APPDATA")
	if env == "" {
		return filepath.Join(home, "AppData/Roaming")
	}
	return env
}

// Get the Linux Data Directory
func GetLinuxDataDir(home string) string {
	env := os.Getenv("XDG_DATA_HOME")
	if env == "" {
		return filepath.Join(home, ".local/share")
	}
	return env
}

// Get the path to '$(DATA_DIR)/typst/packages'
//
// If $TYPST_PACKAGE_PATH is set returns it.
func GetTypstPackagePath() (string, error) {
	typstPackagePath := os.Getenv("TYPST_PACKAGE_PATH")
	if typstPackagePath != "" {
		return typstPackagePath, nil
	}

	goos := runtime.GOOS
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir, err := GetPlatformDataDirectory(goos, home)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "typst", "packages")
	return path, nil
}

func ResolveTargetPath(base, src, dst string) (string, error) {
	relPath, err := filepath.Rel(base, src)
	if err != nil {
		return "", err
	}
	return filepath.Join(dst, relPath), nil

}

var ErrInsufficientPackage = errors.New("both package and version must be specified")

func ResolveUninstallTarget(dataDir string, all bool, ns, pkg, ver string) (string, error) {
	if all {
		if pkg != "" && ver != "" {
			return filepath.Join(dataDir, ns, pkg, ver), nil
		}
		if pkg != "" {
			return filepath.Join(dataDir, ns, pkg), nil
		}
		return filepath.Join(dataDir, ns), nil
	}
	if ver == "" || pkg == "" {
		return "", ErrInsufficientPackage
	}
	return filepath.Join(dataDir, ns, pkg, ver), nil
}
