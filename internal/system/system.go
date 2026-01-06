package system

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	MACOS   string = "darwin"
	WINDOWS string = "windows"
	LINUX   string = "linux"
)

var ErrOperatingSystem = errors.New("unsupported operating system")

// Get the Typst Package Path from a given Operating System
// and a given Home Directory
func GetTypstPath(os string, home string) (string, error) {
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
