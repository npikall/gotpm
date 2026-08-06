package paths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// Size sums the bytes held by the given files and directories. Paths that do
// not exist contribute nothing, so an absent cache directory reports as empty
// rather than as an error.
func Size(targets ...string) (int64, error) {
	var total int64
	for _, target := range targets {
		err := filepath.WalkDir(target, func(_ string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}
				return fmt.Errorf("reading file info: %w", err)
			}
			total += info.Size()
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("measuring %q: %w", target, err)
		}
	}
	return total, nil
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
