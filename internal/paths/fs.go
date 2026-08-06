package paths

import (
	"fmt"
	"io/fs"
	"os"
)

const (
	DirPerm  fs.FileMode = 0o750
	FilePerm fs.FileMode = 0o644
)

// EnsureDir creates path and all necessary parents.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, DirPerm); err != nil {
		return fmt.Errorf("creating directory %q: %w", path, err)
	}
	return nil
}

// IsDir reports whether path is a directory, without following symlinks.
func IsDir(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// WriteFile writes data to path with FilePerm.
func WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, FilePerm); err != nil {
		return fmt.Errorf("could not write file %q: %w", path, err)
	}
	return nil
}
