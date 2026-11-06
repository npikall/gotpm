package compilers //nolint: testpackage // Typst zips only windows builds, which the exported API never unpacks elsewhere

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnpack_FindsTypstExeInAZip(t *testing.T) {
	t.Parallel()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, content := range map[string]string{
		"typst-x86_64-pc-windows-msvc/LICENSE":   "license",
		"typst-x86_64-pc-windows-msvc/typst.exe": "the compiler",
	} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	target := filepath.Join(t.TempDir(), "typst.exe")

	err := unpack(Asset{Name: "typst-x86_64-pc-windows-msvc.zip"}, archive.Bytes(), target)

	require.NoError(t, err)
	content, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "the compiler", string(content))
}
