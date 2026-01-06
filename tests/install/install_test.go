package install_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	i "github.com/npikall/gotpm/internal/install"
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
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s want %s", string(got), string(want))
	}
}
