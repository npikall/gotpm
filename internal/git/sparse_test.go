package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The packages a sparse fork clone stands in for: one is published, the others
// have to survive untouched.
const (
	published = "packages/preview/alpha"
	bystander = "packages/preview/beta"
	othertoo  = "packages/preview/gamma"
)

// TestSparseCommitKeepsUnscopedPackages is the assertion the whole sparse
// scheme rests on. The clone materializes one package directory, so the commit
// is built from an index whose other entries are flagged SkipWorktree and whose
// files are not on disk. If those entries were dropped instead of carried over,
// the commit would delete every other package - a pull request against the
// Typst Universe that removes its index.
func TestSparseCommitKeepsUnscopedPackages(t *testing.T) {
	t.Parallel()
	origin := setupPackagesOrigin(t)
	dst := filepath.Join(t.TempDir(), "fork")

	repo, err := git.SparseClone(dst, "file://"+origin)
	require.NoError(t, err)
	defer repo.Close()

	require.NoError(t, repo.SparseCheckout("main", published))

	assert.DirExists(t, filepath.Join(dst, filepath.FromSlash(published)),
		"the scoped package is materialized")
	assert.NoDirExists(t, filepath.Join(dst, filepath.FromSlash(bystander)),
		"a package outside the scope stays off disk")

	before := treeOf(t, dst, "HEAD")

	writeFile(t, dst, published+"/0.2.0/typst.toml", "name = \"alpha\"\n")
	require.NoError(t, repo.Add(published))
	hash, err := repo.Commit("release: alpha 0.2.0")
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	after := treeOf(t, dst, "HEAD")

	for _, path := range []string{bystander + "/0.1.0/typst.toml", othertoo + "/0.1.0/typst.toml"} {
		assert.Equal(t, before[path], after[path],
			"%s must be carried into the commit unchanged", path)
	}
	assert.Len(t, after, len(before)+1, "the commit adds one file and removes none")
	assert.Contains(t, after, published+"/0.2.0/typst.toml")
}

// TestSparsePushSendsOnlyTheChange covers the other half of the sparse scheme:
// a push works out what to send by walking the new commit against what the
// remote already has. The walk has to prune the packages that did not change
// by comparing subtree hashes - reading through them would mean wanting objects
// a blobless clone deliberately never fetched.
func TestSparsePushSendsOnlyTheChange(t *testing.T) {
	t.Parallel()
	origin := setupPackagesOrigin(t)
	dst := filepath.Join(t.TempDir(), "fork")

	repo, err := git.SparseClone(dst, "file://"+origin)
	require.NoError(t, err)
	defer repo.Close()

	const branch = "alpha-0.2.0"
	require.NoError(t, repo.SetBranchTo(branch, "origin/main"))
	require.NoError(t, repo.SparseCheckout(branch, published))

	writeFile(t, dst, published+"/0.2.0/typst.toml", "name = \"alpha\"\n")
	require.NoError(t, repo.Add(published))
	_, err = repo.Commit("release: alpha 0.2.0")
	require.NoError(t, err)

	require.NoError(t, repo.Push(branch))

	// The origin is the oracle: it has to end up with the new package version
	// and every package that was never materialized in the clone.
	pushed := treeOf(t, origin, branch)
	assert.Contains(t, pushed, published+"/0.2.0/typst.toml")
	assert.Contains(t, pushed, bystander+"/0.1.0/typst.toml")
	assert.Contains(t, pushed, othertoo+"/0.1.0/typst.toml")

	// Pushing again has nothing to send and must not be reported as failure.
	require.NoError(t, repo.Push(branch))
}

// TestSparseCheckoutRescopes covers publishing a second package out of the same
// clone: the previous package's files have to leave the worktree, because the
// index says they are not part of the checkout any more.
func TestSparseCheckoutRescopes(t *testing.T) {
	t.Parallel()
	origin := setupPackagesOrigin(t)
	dst := filepath.Join(t.TempDir(), "fork")

	repo, err := git.SparseClone(dst, "file://"+origin)
	require.NoError(t, err)
	defer repo.Close()

	require.NoError(t, repo.SparseCheckout("main", published))
	require.NoError(t, repo.SparseCheckout("main", bystander))

	assert.DirExists(t, filepath.Join(dst, filepath.FromSlash(bystander)))
	assert.NoDirExists(t, filepath.Join(dst, filepath.FromSlash(published)),
		"the package scoped out is removed from disk")
}

// TestSparseCheckoutOnAbsentPath covers a package's first publish: nothing
// upstream has the directory yet, so scoping to it must not fail.
func TestSparseCheckoutOnAbsentPath(t *testing.T) {
	t.Parallel()
	origin := setupPackagesOrigin(t)
	dst := filepath.Join(t.TempDir(), "fork")

	repo, err := git.SparseClone(dst, "file://"+origin)
	require.NoError(t, err)
	defer repo.Close()

	require.NoError(t, repo.SparseCheckout("main", "packages/preview/brand-new"))
}

// setupPackagesOrigin builds a repository laid out like typst/packages, with
// one version of each of three packages on main.
func setupPackagesOrigin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	for _, pkg := range []string{published, bystander, othertoo} {
		writeFile(t, dir, pkg+"/0.1.0/typst.toml", "dummy\n")
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "initial commit")
	return dir
}

// treeOf returns the blob hash of every file in a revision's tree, keyed by
// slash-separated path, read with the git binary rather than go-git so the
// assertion does not depend on the code under test.
func treeOf(t *testing.T, dir, rev string) map[string]string {
	t.Helper()
	out := gitOut(t, dir, "ls-tree", "-r", rev)
	tree := map[string]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		meta, path, found := strings.Cut(line, "\t")
		require.True(t, found, "unexpected ls-tree line %q", line)
		fields := strings.Fields(meta)
		require.Len(t, fields, 3, "unexpected ls-tree line %q", line)
		tree[path] = fields[2]
	}
	return tree
}

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(path))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	gitOut(t, dir, args...)
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}
