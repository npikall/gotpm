package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

func linuxDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, ".local", "share"), nil
}
