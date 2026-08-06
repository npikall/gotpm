package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// LinuxDataDir returns the data directory used on linux ($XDG_DATA_HOME or
// ~/.local/share).
func LinuxDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, ".local", "share"), nil
}
