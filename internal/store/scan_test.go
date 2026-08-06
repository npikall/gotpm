package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// install creates a version directory inside a store root.
func install(t *testing.T, root, namespace, name, version string) {
	t.Helper()
	require.NoError(t, paths.EnsureDir(filepath.Join(root, namespace, name, version)))
}

// installEditable symlinks a version directory to a source directory.
func installEditable(t *testing.T, root, namespace, name, version string) {
	t.Helper()
	src := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(src, "main.typ"), []byte("")))
	require.NoError(t, paths.EnsureDir(filepath.Join(root, namespace, name)))
	require.NoError(t, os.Symlink(src, filepath.Join(root, namespace, name, version)))
}

func TestScan_Regular(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	install(t, root, "@ns", "pkg", "0.1.0")

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)
	require.Len(t, namespaces, 1)

	assert.Equal(t, "@ns", namespaces[0].Name)
	require.Len(t, namespaces[0].Packages, 1)
	assert.Equal(t, "pkg", namespaces[0].Packages[0].Name)
	require.Len(t, namespaces[0].Packages[0].Versions, 1)
	assert.Equal(t, "0.1.0", namespaces[0].Packages[0].Versions[0].Name)
	assert.False(t, namespaces[0].Packages[0].Versions[0].Editable)
}

func TestScan_Editable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	installEditable(t, root, "@local", "mypkg", "0.2.0")

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)
	require.Len(t, namespaces, 1)

	versions := namespaces[0].Packages[0].Versions
	require.Len(t, versions, 1)
	assert.Equal(t, "0.2.0", versions[0].Name)
	assert.True(t, versions[0].Editable, "a symlinked version is an editable install")
}

func TestScan_MixedVersions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	install(t, root, "@ns", "pkg", "0.1.0")
	installEditable(t, root, "@ns", "pkg", "0.2.0")

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)

	versions := namespaces[0].Packages[0].Versions
	require.Len(t, versions, 2)
	assert.False(t, versions[0].Editable, "0.1.0 is a copy")
	assert.True(t, versions[1].Editable, "0.2.0 is a symlink")
}

func TestScan_EmptyDir(t *testing.T) {
	t.Parallel()
	namespaces, err := store.At(t.TempDir()).Scan()
	require.NoError(t, err)
	assert.Empty(t, namespaces)
}

func TestScan_MissingRoot(t *testing.T) {
	t.Parallel()
	namespaces, err := store.At(filepath.Join(t.TempDir(), "nope")).Scan()
	require.NoError(t, err, "an absent store holds nothing, which is not a failure")
	assert.Empty(t, namespaces)
}

func TestScan_NonDirInNamespaceSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	install(t, root, "@ns", "pkg", "0.1.0")
	require.NoError(t, paths.WriteFile(filepath.Join(root, "@ns", "not-a-pkg"), []byte("")))

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)
	assert.Len(t, namespaces[0].Packages, 1)
}

func TestScan_PackageWithNoVersionsExcluded(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, paths.EnsureDir(filepath.Join(root, "@ns", "empty-pkg")))

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)
	assert.Empty(t, namespaces, "a namespace whose packages hold no versions is not listed")
}

func TestScan_NonSemverVersionIsListed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	install(t, root, "@ns", "pkg", "not-a-version")

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	assert.Equal(t, "not-a-version", namespaces[0].Packages[0].Versions[0].Name,
		"scanning reports what is on disk, whatever it is named")
}

func TestScan_SortedAlphabetically(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	install(t, root, "zeta", "zpkg", "0.2.0")
	install(t, root, "zeta", "zpkg", "0.1.0")
	install(t, root, "zeta", "apkg", "1.0.0")
	install(t, root, "alpha", "pkg", "1.0.0")

	namespaces, err := store.At(root).Scan()
	require.NoError(t, err)

	require.Len(t, namespaces, 2)
	assert.Equal(t, "alpha", namespaces[0].Name)
	assert.Equal(t, "zeta", namespaces[1].Name)

	zeta := namespaces[1].Packages
	require.Len(t, zeta, 2)
	assert.Equal(t, "apkg", zeta[0].Name)
	assert.Equal(t, "zpkg", zeta[1].Name)

	assert.Equal(t, "0.1.0", zeta[1].Versions[0].Name)
	assert.Equal(t, "0.2.0", zeta[1].Versions[1].Name)
}

func TestExists(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	assert.True(t, store.At(root).Exists())
	assert.False(t, store.At(filepath.Join(root, "nope")).Exists())
}
