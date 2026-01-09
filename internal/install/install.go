package install

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

type CopyWorker struct {
	Src string
	Dst string
}

func (c CopyWorker) Copy() {
	err := os.MkdirAll(filepath.Dir(c.Dst), 0750)
	if err != nil {
		log.Println(err)
		return
	}
	err = CopyFile(c.Src, c.Dst)
	if err != nil {
		log.Println(err)
		return
	}
}

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
