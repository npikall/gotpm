package paths

import (
	"fmt"
	"os"
)

// WindowsDataDir returns the data directory used on windows (%APPDATA%).
func WindowsDataDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("%w: %%APPDATA%% is not set", ErrDataDirNotResolvable)
	}
	return appData, nil
}
