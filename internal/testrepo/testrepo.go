// Package testrepo builds git repositories holding typst packages, so tests
// can resolve, clone and install real packages over file:// instead of
// reaching for the network or pre-seeding a cache by hand.
//
// It exists to be shared: resolving a dependency graph, adding it to a project
// and syncing a checkout are all tested against the same kind of fixture, and
// building that fixture is more code than any one of those tests.
package testrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/require"
)

// Isolate points gotpm's data and package directories at temporary ones and
// returns the package directory root. A test calling it cannot be parallel:
// t.Setenv is process-wide.
func Isolate(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("HOME", data)
	t.Setenv("APPDATA", data)

	packages := filepath.Join(data, "typst-packages")
	t.Setenv("TYPST_PACKAGE_PATH", packages)
	return packages
}

// Project writes a minimal typst.toml into a new directory, makes it the
// working directory, and returns it. This is the project the dependency
// commands operate on.
func Project(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	content := fmt.Sprintf("[package]\nname = %q\nversion = \"0.1.0\"\nentrypoint = \"main.typ\"\n", name)
	require.NoError(t, paths.WriteFile(filepath.Join(dir, manifest.FileName), []byte(content)))
	t.Chdir(dir)
	return dir
}

// Package is a git repository holding one typst package.
type Package struct {
	t       *testing.T
	dir     string
	repo    *git.Repository
	name    string
	version string
	hash    string
	builds  int
}

// New creates an empty repository for a package. Nothing is committed until
// Release is called, because a dependency has to exist before the package that
// depends on it can lock the commit it sits at.
func New(t *testing.T, name, version string) *Package {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	require.NoError(t, paths.EnsureDir(dir))
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	return &Package{t: t, dir: dir, repo: repo, name: name, version: version}
}

// Dir is the working tree of the repository.
func (p *Package) Dir() string { return p.dir }

// URL is the address the package is reachable at.
func (p *Package) URL() string { return "file://" + p.dir }

// Import is the statement typst source imports the package with.
func (p *Package) Import() string {
	return "@" + manifest.Namespace + "/" + p.name + ":" + p.version
}

// Tag is the release tag of the version.
func (p *Package) Tag() string { return "v" + p.version }

// Hash is the commit of the most recent release.
func (p *Package) Hash() string { return p.hash }

// Release commits the package, declaring the given dependencies and locking
// each of them to the commit it currently sits at.
func (p *Package) Release(deps ...*Package) *Package {
	p.t.Helper()
	declared := make([]string, 0, len(deps))
	lock := lockfile.New()
	for _, dep := range deps {
		declared = append(declared, dep.Import())
		lock.Upsert(dep.LockEntry())
	}
	return p.ReleaseWith(declared, lock)
}

// ReleaseWith commits the package declaring exactly declared and shipping
// lock. A nil lock ships no gotpm.lock at all, which is what a package whose
// author forgot to commit one looks like.
func (p *Package) ReleaseWith(declared []string, lock *lockfile.Lock) *Package {
	t := p.t
	t.Helper()

	p.builds++
	lib := fmt.Sprintf("#let name = %q\n#let build = %d\n", p.name, p.builds)
	require.NoError(t, paths.WriteFile(filepath.Join(p.dir, manifest.FileName), p.manifest(declared)))
	require.NoError(t, paths.WriteFile(filepath.Join(p.dir, "lib.typ"), []byte(lib)))
	if lock != nil {
		require.NoError(t, lockfile.Save(p.dir, lock))
	}

	wt, err := p.repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))
	hash, err := wt.Commit("release "+p.version, &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	_ = p.repo.DeleteTag(p.Tag())
	_, err = p.repo.CreateTag(p.Tag(), hash, nil)
	require.NoError(t, err)

	p.hash = hash.String()
	return p
}

// LockEntry is how a dependant, or a project, pins this package.
func (p *Package) LockEntry() lockfile.Entry {
	require.NotEmpty(p.t, p.hash, "%s must be released before something can depend on it", p.name)
	return lockfile.Entry{
		Import:    p.Import(),
		Name:      p.name,
		Version:   p.version,
		Namespace: manifest.Namespace,
		URL:       p.URL(),
		Revision:  p.Tag(),
		Hash:      p.hash,
	}
}

func (p *Package) manifest(declared []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "[package]\nname = %q\nversion = %q\nentrypoint = \"lib.typ\"\n", p.name, p.version)
	if len(declared) > 0 {
		b.WriteString("\n[tool.gotpm]\ndependencies = [\n")
		for _, dep := range declared {
			fmt.Fprintf(&b, "  %q,\n", dep)
		}
		b.WriteString("]\n")
	}
	return []byte(b.String())
}
