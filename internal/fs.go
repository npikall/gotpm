package internal

import (
	"fmt"
	"io/fs"
	"os"
)

const (
	DirPerm  fs.FileMode = 0o750
	FilePerm fs.FileMode = 0o644
)

// WriteFile writes data to path with FilePerm.
func WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, FilePerm); err != nil {
		return fmt.Errorf("could not write file %q: %w", path, err)
	}
	return nil
}
