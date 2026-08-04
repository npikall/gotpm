package paths

import (
	"fmt"
	"os"
)

func windowsDataDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("%w: %%APPDATA%% is not set", ErrDataDirNotResolvable)
	}
	return appData, nil
}
