package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// DarwinDataDir returns the data directory used on macOS.
func DarwinDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}
