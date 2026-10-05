package testrepo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseDeclaresAndLocksItsDependencies(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	dep := testrepo.New(t, "dep", "0.1.0").Release()
	pkg := testrepo.New(t, "pkg", "1.0.0").Release(dep)

	m, err := manifest.LoadFrom(pkg.Dir())
	require.NoError(t, err)
	assert.Equal(t, []string{dep.Import()}, m.Tool.Gotpm.Dependencies)

	lock, err := lockfile.Load(pkg.Dir())
	require.NoError(t, err)
	entry, ok := lock.Get(dep.Import())
	require.True(t, ok)
	assert.Equal(t, dep.Hash(), entry.Hash)

	repo, err := git.PlainOpen(pkg.Dir())
	require.NoError(t, err)
	tag, err := repo.Tag(pkg.Tag())
	require.NoError(t, err, "a standalone package's release is tagged")
	assert.Equal(t, pkg.Hash(), tag.Hash().String())
}

func TestReleaseWithoutLockShipsNone(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	pkg := testrepo.New(t, "pkg", "1.0.0").ReleaseWith(nil, nil)

	assert.NoFileExists(t, filepath.Join(pkg.Dir(), lockfile.FileName))
}

func TestMonorepoPackagesAreUntagged(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	mono := testrepo.NewMonorepo(t, "mono")
	pkg := mono.Package("packages/common", "common", "0.1.0").Release()

	assert.Equal(t, "HEAD", pkg.Tag())
	assert.Equal(t, mono.URL()+"//packages/common", pkg.URL())
	_, err := os.Stat(filepath.Join(pkg.Dir(), "packages", "common", manifest.FileName))
	require.NoError(t, err)
}

func TestProjectBecomesTheWorkingDirectory(t *testing.T) { //nolint: paralleltest // t.Chdir
	dir := testrepo.Project(t, "doc")

	m, err := manifest.Load()
	require.NoError(t, err)
	assert.Equal(t, "doc", m.Package.Name)
	assert.DirExists(t, dir)
}

func TestReleaseFontsDeclaresAndPinsThem(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	pin := lockfile.Font{Family: "Lato", Source: "google-fonts", Hash: "abc"}

	p := testrepo.New(t, "thesis", "1.0.0").ReleaseFonts(pin)

	m, err := manifest.LoadFrom(p.Dir())
	require.NoError(t, err)
	assert.Equal(t, []string{"Lato"}, m.Fonts())
	lock, err := lockfile.Load(p.Dir())
	require.NoError(t, err)
	got, ok := lock.GetFont("Lato")
	require.True(t, ok)
	assert.True(t, got.Direct)
}
