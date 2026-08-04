package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

func darwinDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}
