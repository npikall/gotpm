package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/require"
)

// writeManifest creates a package directory containing the given typst.toml.
func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(content), paths.FilePerm))
	return dir
}

// check fails the surrounding test setup on error.
func check(e error) {
	if e != nil {
		panic(e)
	}
}
