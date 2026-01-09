package install_test

import (
	"os"
	"path/filepath"
	"testing"

	i "github.com/npikall/gotpm/internal/install"
	"github.com/stretchr/testify/assert"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	want := []byte("Hello World")
	os.WriteFile(src, want, 0644)

	// Actual tested function
	if err := i.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	got, _ := os.ReadFile(dst)
	assert.Equal(t, want, got, "got %s want %s", string(got), string(want))
}

func TestResolveTargetPath(t *testing.T) {
	cwd := "src/dir/"
	src := "src/dir/main.go"
	dst := "dst/dir/"
	want := "dst/dir/main.go"

	got, err := i.ResolveTargetPath(cwd, src, dst)
	assert.NoError(t, err, "expected no error, but got one")
	assert.Equal(t, want, got, "got %s want %s", got, want)
}
