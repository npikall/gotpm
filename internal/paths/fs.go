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

// Remove deletes path from disk. A symlink is removed itself rather than
// followed, so the directory it points at is left alone; anything else is
// removed with its contents.
func Remove(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("checking target %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("could not remove %q: %w", path, err)
		}
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("could not remove-all %q: %w", path, err)
	}
	return nil
}

// WriteFile writes data to path with FilePerm.
func WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, FilePerm); err != nil {
		return fmt.Errorf("could not write file %q: %w", path, err)
	}
	return nil
}
