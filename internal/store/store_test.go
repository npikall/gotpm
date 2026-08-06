package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ref(t *testing.T) pkg.Ref {
	t.Helper()
	r, err := pkg.New("local", "my-pkg", "0.1.0")
	require.NoError(t, err)
	return r
}

// sourcePackage creates a package directory with one file in it.
func sourcePackage(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	// The temp dir may sit behind a symlink (/tmp on macOS), which would make
	// the symlink assertions compare different spellings of the same path.
	dir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.NoError(t, paths.WriteFile(filepath.Join(dir, "lib.typ"), []byte(content)))
	return dir
}

func TestAt_LaysOutByNamespaceNameVersion(t *testing.T) {
	t.Parallel()
	s := store.At("/data")
	assert.Equal(t, filepath.Join("/data", "local", "my-pkg", "0.1.0"), s.Dir(ref(t)))
	assert.Equal(t, "/data", s.Root())
}

func TestOpen_OverrideIsTheDestination(t *testing.T) {
	t.Setenv(paths.InstallDirEnvVar, "")
	override := t.TempDir()

	s, err := store.Open(override)
	require.NoError(t, err)
	assert.Equal(t, override, s.Dir(ref(t)), "an overridden store must not append sub-directories")
}

func TestOpen_EnvOverrideIsTheDestination(t *testing.T) {
	override := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, override)

	s, err := store.Open("")
	require.NoError(t, err)
	assert.Equal(t, override, s.Dir(ref(t)))
}

func TestOpen_WithoutOverrideKeepsTheLayout(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(paths.InstallDirEnvVar, "")
	t.Setenv("TYPST_PACKAGE_PATH", tmp)

	s, err := store.Open("")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, "local", "my-pkg", "0.1.0"), s.Dir(ref(t)))
}

func TestInstall(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)

	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))

	content, err := os.ReadFile(filepath.Join(s.Dir(r), "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, "v1", string(content))
}

func TestLink(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	src := sourcePackage(t, "content")

	require.NoError(t, s.Link(r, src))

	info, err := os.Lstat(s.Dir(r))
	require.NoError(t, err)
	assert.NotEqual(t, os.FileMode(0), info.Mode()&os.ModeSymlink, "an editable install must be a symlink")

	target, err := os.Readlink(s.Dir(r))
	require.NoError(t, err)
	assert.Equal(t, src, target)

	content, err := os.ReadFile(filepath.Join(s.Dir(r), "lib.typ"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(content), "source contents must be reachable through the link")
}

func TestLink_RelativeSourceIsResolved(t *testing.T) { //nolint: paralleltest
	src := sourcePackage(t, "content")
	t.Chdir(src)

	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Link(r, "."))

	target, err := os.Readlink(s.Dir(r))
	require.NoError(t, err)
	assert.Equal(t, src, target)
}

func TestHas(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)

	assert.False(t, s.Has(r), "nothing is installed yet")

	require.NoError(t, s.Install(r, sourcePackage(t, "x")))
	assert.True(t, s.Has(r))
}

func TestHas_DanglingSymlinkCounts(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)

	src := sourcePackage(t, "x")
	require.NoError(t, s.Link(r, src))
	require.NoError(t, os.RemoveAll(src))

	assert.True(t, s.Has(r), "an editable install whose source is gone is still installed")
}

func TestRemove(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "x")))

	require.NoError(t, s.Remove(r))
	assert.False(t, s.Has(r))
}

func TestRemove_LeavesTheLinkTargetAlone(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	src := sourcePackage(t, "content")
	require.NoError(t, s.Link(r, src))

	require.NoError(t, s.Remove(r))

	assert.False(t, s.Has(r))
	assert.FileExists(t, filepath.Join(src, "lib.typ"), "removing an editable install must not touch the source")
}

func TestRemove_MissingPackageIsNotAnError(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	assert.NoError(t, s.Remove(ref(t)))
}
