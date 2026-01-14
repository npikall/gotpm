package uninstall

import (
	"errors"
	"path/filepath"
)

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
