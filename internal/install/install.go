package install

import (
	"io"
	"os"
	"path/filepath"
)

// Copy a file from src to dst
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Sync()
	return err
}

func ResolveTargetPath(base, src, dst string) (string, error) {
	relPath, err := filepath.Rel(base, src)
	if err != nil {
		return "", err
	}
	return filepath.Join(dst, relPath), nil

}
