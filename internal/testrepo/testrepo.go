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
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/require"
)

const (
	packagesRelPath = "typst-packages"
	configRelPath   = "config"
)

// Packages is the package store inside an isolated root.
func Packages(root string) string { return filepath.Join(root, packagesRelPath) }

// GitConfigPath is the global git config inside an isolated root. Nothing
// creates it: git reads a missing config as an empty one, so isolation holds
// whether or not a test has written anything there.
func GitConfigPath(root string) string { return filepath.Join(root, "gitconfig") }

// WriteGitIdentity gives the isolated root a git identity, for the tests that
// exercise gotpm committing on the user's behalf. It is written rather than
// assumed because isolation removed the real one, and a machine with no
// identity is a machine those commands correctly refuse to work on.
func WriteGitIdentity(t *testing.T, root string) {
	t.Helper()
	content := "[user]\n\tname = test\n\temail = test@example.com\n"
	require.NoError(t, paths.WriteFile(GitConfigPath(root), []byte(content)))
}

// Env is what isolation consists of, as KEY=VALUE pairs rooted at root: every
// variable gotpm or go-git reads to find state on the machine, pointed
// somewhere disposable.
//
// It is the whole list, not an addition to the ambient environment. A test
// running gotpm as a subprocess passes exactly these (plus PATH), so nothing
// the developer or the runner exports can reach it — GITHUB_TOKEN and
// SSH_AUTH_SOCK above all, which would otherwise hand a publish test real push
// credentials.
func Env(root string) []string {
	return []string{
		"HOME=" + root,
		"APPDATA=" + root,
		"XDG_DATA_HOME=" + root,
		"XDG_CONFIG_HOME=" + filepath.Join(root, configRelPath),
		"TYPST_PACKAGE_PATH=" + Packages(root),
		// Empty reads as unset everywhere gotpm consults it.
		"GOTPM_INSTALL_DIR=",
		"GIT_CONFIG_GLOBAL=" + GitConfigPath(root),
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

// Isolate applies Env to the running process for the duration of t, so a test
// calling gotpm in-process never touches the developer's real clone cache,
// config or installed packages. It returns the root of the package store.
//
// A test calling this cannot be parallel: t.Setenv is process-wide.
func Isolate(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, pair := range Env(root) {
		key, value, _ := strings.Cut(pair, "=")
		t.Setenv(key, value)
	}
	return Packages(root)
}

// ProjectAt writes a minimal typst.toml into a new directory and returns it,
// without changing where the test is running. This is the project the
// dependency commands operate on; a test driving gotpm as a subprocess points
// its working directory here instead of moving its own.
func ProjectAt(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	content := fmt.Sprintf("[package]\nname = %q\nversion = \"0.1.0\"\nentrypoint = \"main.typ\"\n", name)
	require.NoError(t, paths.WriteFile(filepath.Join(dir, manifest.FileName), []byte(content)))
	return dir
}

// Project is ProjectAt, made the working directory. A test calling this cannot
// be parallel: t.Chdir is process-wide.
func Project(t *testing.T, name string) string {
	t.Helper()
	dir := ProjectAt(t, name)
	t.Chdir(dir)
	return dir
}

// initRepo creates a repository on main that can be committed to whatever the
// machine running the test is configured to do.
//
// go-git refuses to commit at all when commit.gpgSign is set without a signer,
// and it reads the user's global config unless the process was told otherwise.
// A fixture that only works for developers who do not sign their commits is
// not a fixture, so the setting is turned off in the repository itself — the
// same thing publishing does to a Fork Clone, and for the same reason.
//
// The branch is named explicitly because go-git still defaults to master,
// which is not what the repositories gotpm is pointed at in the world are
// called.
func initRepo(t *testing.T, dir string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(dir, false, git.WithDefaultBranch(plumbing.Main))
	require.NoError(t, err)

	cfg, err := repo.Config()
	require.NoError(t, err)
	cfg.Raw.Section("commit").SetOption("gpgsign", "false")
	require.NoError(t, repo.SetConfig(cfg))
	return repo
}

// Fork creates a bare repository standing in for a fork of the Typst Universe
// package repository, and returns the URL it is reachable at.
//
// It is seeded with a commit, because that is what makes it a fork rather than
// an empty repository: publishing positions a branch before it scopes a
// worktree, and a repository with no commits has no branch to position against.
func Fork(t *testing.T) string {
	t.Helper()
	seed := filepath.Join(t.TempDir(), "seed")
	require.NoError(t, paths.EnsureDir(seed))
	repo := initRepo(t, seed)

	require.NoError(t, paths.WriteFile(filepath.Join(seed, "README.md"), []byte("# packages\n")))
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))
	_, err = wt.Commit("seed", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	bare := filepath.Join(t.TempDir(), "fork.git")
	_, err = git.PlainClone(bare, &git.CloneOptions{URL: "file://" + seed, Bare: true})
	require.NoError(t, err)
	return "file://" + bare
}

// Package is a git repository holding one typst package.
type Package struct {
	t       *testing.T
	dir     string
	repo    *git.Repository
	name    string
	version string
	hash    string
	// builds counts releases, so committing the same version twice still
	// changes a file and produces a second commit.
	builds int
}

// New creates an empty repository for a package. Nothing is committed until
// Release is called, because a dependency has to exist before the package that
// depends on it can lock the commit it sits at.
func New(t *testing.T, name, version string) *Package {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	require.NoError(t, paths.EnsureDir(dir))
	return &Package{t: t, dir: dir, repo: initRepo(t, dir), name: name, version: version}
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

	// Releasing the same version again moves the tag, which is exactly the
	// upstream mistake gotpm's pinning is there to survive.
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
