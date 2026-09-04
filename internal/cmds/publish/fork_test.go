package publish_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/publish"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pkgDir = "packages/preview/foo"

// TestMain detaches every git invocation in this package from the developer's
// own git configuration, which could otherwise sign commits, rename the
// initial branch or fail for want of an identity.
func TestMain(m *testing.M) {
	os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	os.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	os.Exit(m.Run())
}

// TestCheckoutPackageBranchSparseScope guards against the bug where checking
// out a package that has never been published before widens the sparse
// checkout to the entire packages/preview namespace (1000+ unrelated package
// directories in the real typst/packages repo) instead of staying scoped to
// just the one package being published.
func TestCheckoutPackageBranchSparseScope(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{
		"packages/preview/existing-pkg",
		"packages/preview/other-pkg-1",
		"packages/preview/other-pkg-2",
	})
	forkPath := cloneFork(t, origin)

	branchExisted, err := publish.CheckoutPackageBranch(
		testLogger(), forkPath, "new-pkg-0.1.0", "packages/preview/new-pkg",
	)
	require.NoError(t, err)
	assert.False(t, branchExisted)

	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/existing-pkg"))
	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/other-pkg-1"))
	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/other-pkg-2"))
}

// TestCheckoutPackageBranchSetsUpstream covers the first publish of a package:
// nothing to fetch, and the branch must still end up tracking origin/<branch>
// so that a manual `git push` after `gotpm publish --local` works. Left to
// git, `checkout -B <branch> origin/main` sets the upstream to origin/main
// instead, and a bare `git push` then offers to push onto the fork's main.
func TestCheckoutPackageBranchSetsUpstream(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	forkPath := cloneFork(t, origin)

	branchExisted, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.False(t, branchExisted, "a package absent from the fork has not been published")

	assert.Equal(t, "origin", gitOut(t, forkPath, "config", "branch.foo-0.1.0.remote"))
	assert.Equal(t, "refs/heads/foo-0.1.0", gitOut(t, forkPath, "config", "branch.foo-0.1.0.merge"))
	assert.True(t, gitcli.TracksOwnBranch(forkPath, "foo-0.1.0"))
}

// TestCheckoutPackageBranchRepairsUpstream covers a branch created by an older
// gotpm, which left it tracking origin/main. Setting the upstream on the
// existing-branch path too is what repairs those on the next publish, rather
// than leaving them broken until the fork clone is deleted.
func TestCheckoutPackageBranchRepairsUpstream(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	forkPath := cloneFork(t, origin)

	require.NoError(t, gitcli.CheckoutNewBranch(forkPath, "foo-0.1.0", "origin/main"))
	require.Equal(t, "refs/heads/main", gitOut(t, forkPath, "config", "branch.foo-0.1.0.merge"),
		"precondition: git points the new branch at origin/main")

	branchExisted, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.True(t, branchExisted)
	assert.True(t, gitcli.TracksOwnBranch(forkPath, "foo-0.1.0"))
}

// TestEnsureForkRepoFetchesMain covers the stale base: a fork cloned once and
// reused for months cuts new branches from whatever main was at clone time
// unless the clone is fetched first.
func TestEnsureForkRepoFetchesMain(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	forkPath := cloneFork(t, origin)

	commitOn(t, origin, "main", "second commit")
	newTip := gitOut(t, origin, "rev-parse", "main")

	require.NoError(t, publish.EnsureForkRepo(testLogger(), cloneURL(origin), forkPath))
	assert.Equal(t, newTip, gitOut(t, forkPath, "rev-parse", "origin/main"))

	_, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.Equal(t, newTip, gitOut(t, forkPath, "rev-parse", "HEAD"),
		"a new branch is cut from the fetched main, not the one pinned at clone time")
}

// TestCheckoutPackageBranchAdoptsForkBranch covers a package branch that
// exists on the fork but not locally - published from another machine, or from
// a fork clone that has since been deleted. Cutting it from origin/main here
// would produce a branch sharing no commit with the fork's, and the push at
// the end of a publish would be rejected as a non-fast-forward.
func TestCheckoutPackageBranchAdoptsForkBranch(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	forkTip := gitOut(t, origin, "rev-parse", "foo-0.1.0")
	forkPath := cloneFork(t, origin)

	require.False(t, gitcli.BranchExists(forkPath, "foo-0.1.0"), "precondition: not cloned locally")

	branchExisted, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.True(t, branchExisted, "a branch on the fork counts as published")
	assert.Equal(t, forkTip, gitOut(t, forkPath, "rev-parse", "HEAD"))
}

// TestCheckoutPackageBranchFastForwards covers a local branch left behind by
// commits made to the fork's branch elsewhere.
func TestCheckoutPackageBranchFastForwards(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	forkPath := cloneFork(t, origin)

	_, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)

	commitOn(t, origin, "foo-0.1.0", "fix: typo")
	newTip := gitOut(t, origin, "rev-parse", "foo-0.1.0")

	branchExisted, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.True(t, branchExisted)
	assert.Equal(t, newTip, gitOut(t, forkPath, "rev-parse", "HEAD"))
}

// TestCheckoutPackageBranchDivergedResetsOntoFork covers the state where the
// local package branch and the fork's branch have both moved on. The fork
// clone is a staging area: every commit on a package branch is a copy of the
// package's working tree, which the publish run about to follow re-copies and
// re-commits. Resetting onto the fork therefore keeps the edits made there -
// a maintainer-requested typst.toml fix, say - and loses nothing that is not
// regenerated.
func TestCheckoutPackageBranchDivergedResetsOntoFork(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	forkPath := cloneFork(t, origin)

	_, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)

	commitOn(t, origin, "foo-0.1.0", "fix: made upstream")
	forkTip := gitOut(t, origin, "rev-parse", "foo-0.1.0")
	runGit(t, forkPath, "commit", "-q", "--allow-empty", "-m", "fix: made locally")

	branchExisted, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.True(t, branchExisted)
	assert.Equal(t, forkTip, gitOut(t, forkPath, "rev-parse", "HEAD"),
		"a diverged branch is reset onto the fork's tip")
}

// TestCheckoutPackageBranchAheadKeepsLocalCommit covers `gotpm publish
// --local`: a commit staged locally and deliberately not pushed yet. The fork
// has not moved, so there is no divergence and nothing to reconcile - the
// reset the diverged case performs must not reach this one.
func TestCheckoutPackageBranchAheadKeepsLocalCommit(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	forkPath := cloneFork(t, origin)

	_, err := publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)

	runGit(t, forkPath, "commit", "-q", "--allow-empty", "-m", "fix: staged by publish --local")
	localTip := gitOut(t, forkPath, "rev-parse", "HEAD")

	_, err = publish.CheckoutPackageBranch(testLogger(), forkPath, "foo-0.1.0", pkgDir)
	require.NoError(t, err)
	assert.Equal(t, localTip, gitOut(t, forkPath, "rev-parse", "HEAD"),
		"a branch merely ahead of the fork keeps its unpushed commit")
}

func testLogger() *log.Logger {
	return log.New(io.Discard)
}

// setupOriginRepo creates a local git repo on branch main containing an empty
// marker file under each of pkgDirs, standing in for a subset of
// typst/packages' packages/preview/* directories. It returns its path; pass it
// through cloneURL to clone from it.
func setupOriginRepo(t *testing.T, pkgDirs []string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	for _, d := range pkgDirs {
		full := filepath.Join(dir, filepath.FromSlash(d))
		require.NoError(t, os.MkdirAll(full, 0o755))
		toml := filepath.Join(full, "typst.toml")
		require.NoError(t, os.WriteFile(toml, []byte("dummy"), 0o644))
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "initial commit")
	return dir
}

// commitOn adds an empty commit to branch of the repo at dir, creating the
// branch off main when it does not exist yet, and leaves main checked out.
func commitOn(t *testing.T, dir, branch, msg string) {
	t.Helper()
	runGit(t, dir, "checkout", "-q", "-B", branch, branch2base(t, dir, branch))
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", msg)
	runGit(t, dir, "checkout", "-q", "main")
}

func branch2base(t *testing.T, dir, branch string) string {
	t.Helper()
	if gitcli.BranchExists(dir, branch) {
		return branch
	}
	return "main"
}

// cloneURL returns the URL to clone dir from. A plain path makes git ignore
// --depth and --filter ("--depth is ignored in local clones"), which would
// leave the test clone with the full +refs/heads/*:refs/remotes/origin/*
// refspec instead of the single-branch one production gets - and every
// assertion about fetching package branches would then hold for the wrong
// reason.
func cloneURL(dir string) string {
	return "file://" + dir
}

// cloneFork clones dir the way a publish does and configures an identity, so
// that commits made in the clone during a test succeed.
func cloneFork(t *testing.T, origin string) string {
	t.Helper()
	forkPath := t.TempDir()
	require.NoError(t, gitcli.Clone(cloneURL(origin), forkPath))
	runGit(t, forkPath, "config", "user.email", "test@test.com")
	runGit(t, forkPath, "config", "user.name", "test")
	return forkPath
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
}

// gitOut runs git and returns its trimmed standard output.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	require.NoErrorf(t, err, "git %v", args)
	return strings.TrimSpace(string(out))
}
