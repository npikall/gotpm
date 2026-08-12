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
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/npikall/gotpm/internal/cmds/publish"
	"github.com/npikall/gotpm/internal/git"
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

// TestForkCloneHasNoWorktree is the load-bearing property of the whole design:
// publishing writes objects, never files. A worktree would have to be
// materialized from a blobless clone, and go-git cannot do that - it has no
// promisor support, so any checkout of a filtered clone fails on the blobs the
// filter left behind.
func TestForkCloneHasNoWorktree(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{
		"packages/preview/existing-pkg",
		"packages/preview/other-pkg-1",
	})
	forkPath := filepath.Join(t.TempDir(), "fork")

	fork, err := publish.EnsureForkRepo(testLogger(), cloneURL(origin), forkPath)
	require.NoError(t, err)
	require.True(t, fork.HasMain())

	entries, err := os.ReadDir(forkPath)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the clone holds .git and nothing else")
	assert.Equal(t, ".git", entries[0].Name())
}

// TestPreparePackageBranchSetsUpstream covers the first publish of a package:
// nothing to fetch, and the branch must still end up tracking origin/<branch>
// so that a manual `git push` after `gotpm publish --local` works, rather than
// offering to push onto the fork's main.
func TestPreparePackageBranchSetsUpstream(t *testing.T) {
	t.Parallel()
	fork := cloneFork(t, setupOriginRepo(t, []string{pkgDir}))

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.False(t, base.Existed, "a package absent from the fork has not been published")

	assert.Equal(t, "origin", gitOut(t, fork.Path(), "config", "branch.foo-0.1.0.remote"))
	assert.Equal(t, "refs/heads/foo-0.1.0", gitOut(t, fork.Path(), "config", "branch.foo-0.1.0.merge"))
	assert.True(t, fork.TracksOwnBranch("foo-0.1.0"))
}

// TestPreparePackageBranchBasesOnMain covers the first publish cutting its
// branch from the fork's main, so the resulting pull request carries one commit
// rather than reinventing the repository.
func TestPreparePackageBranchBasesOnMain(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.Equal(t, gitOut(t, origin, "rev-parse", "main"), base.Commit.String())
}

// TestEnsureForkRepoFetchesMain covers the stale base: a fork cloned once and
// reused for months cuts new branches from whatever main was at clone time
// unless the clone is fetched first.
func TestEnsureForkRepoFetchesMain(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	fork := cloneFork(t, origin)

	commitOn(t, origin, "main", "second commit")
	newTip := gitOut(t, origin, "rev-parse", "main")

	fork, err := publish.EnsureForkRepo(testLogger(), cloneURL(origin), fork.Path())
	require.NoError(t, err)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.Equal(t, newTip, base.Commit.String(),
		"a new branch is cut from the fetched main, not the one pinned at clone time")
}

// TestEnsureForkRepoReclonesIncomplete covers a clone directory left behind by
// an interrupted first publish. Reusing it would leave every later publish
// building on a repository with no origin/main.
func TestEnsureForkRepoReclonesIncomplete(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	forkPath := filepath.Join(t.TempDir(), "fork")
	require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".git"), 0o755))

	fork, err := publish.EnsureForkRepo(testLogger(), cloneURL(origin), forkPath)
	require.NoError(t, err)
	assert.True(t, fork.HasMain())
}

// TestPreparePackageBranchAdoptsForkBranch covers a package branch that exists
// on the fork but not locally - published from another machine, or from a fork
// clone that has since been deleted. Basing on main here would produce a branch
// sharing no commit with the fork's, and the push would be rejected as a
// non-fast-forward.
func TestPreparePackageBranchAdoptsForkBranch(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	forkTip := gitOut(t, origin, "rev-parse", "foo-0.1.0")
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.True(t, base.Existed, "a branch on the fork counts as published")
	assert.Equal(t, forkTip, base.Commit.String())
}

// TestPreparePackageBranchFastForwards covers a local branch left behind by
// commits made to the fork's branch elsewhere.
func TestPreparePackageBranchFastForwards(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	require.NoError(t, fork.SetBranch("foo-0.1.0", base.Commit))

	commitOn(t, origin, "foo-0.1.0", "fix: typo")
	newTip := gitOut(t, origin, "rev-parse", "foo-0.1.0")

	base, err = publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.True(t, base.Existed)
	assert.Equal(t, newTip, base.Commit.String())
}

// TestPreparePackageBranchKeepsUnpushedLocal covers `gotpm publish --local`
// followed by another publish: the local branch is ahead of the fork, and
// rewinding it to the fork's tip would silently discard the commit the user
// deliberately has not pushed yet.
func TestPreparePackageBranchKeepsUnpushedLocal(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	local := commitPackage(t, fork, base, "release: foo 0.1.0")

	base, err = publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	assert.Equal(t, local.String(), base.Commit.String())
}

// TestPreparePackageBranchDivergedAborts covers the one state that cannot be
// reconciled without choosing on the user's behalf: both sides have moved.
// Resetting to the fork would discard the local commit, and merging could leave
// the clone mid-conflict.
func TestPreparePackageBranchDivergedAborts(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir})
	commitOn(t, origin, "foo-0.1.0", "release: foo 0.1.0")
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	local := commitPackage(t, fork, base, "fix: made locally")

	commitOn(t, origin, "foo-0.1.0", "fix: made upstream")

	_, err = publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.ErrorIs(t, err, publish.ErrForkBranchDiverged)
	assert.Equal(t, local.String(), gitOut(t, fork.Path(), "rev-parse", "foo-0.1.0"),
		"a refused publish leaves the branch where it was")
}

// TestCommitWritesPackageIntoFork checks the commit the fork ends up with:
// the package's files land under their version directory, and every other
// package in the fork is carried over untouched.
func TestCommitWritesPackageIntoFork(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir, "packages/preview/other"})
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	commitPackage(t, fork, base, "release: foo 0.1.0")

	assert.Equal(t, "content",
		gitOut(t, fork.Path(), "show", "foo-0.1.0:packages/preview/foo/0.1.0/lib.typ"))
	assert.Equal(t, "dummy",
		gitOut(t, fork.Path(), "show", "foo-0.1.0:packages/preview/other/typst.toml"),
		"an unrelated package is carried over by hash, not rewritten")
	assert.Equal(t, gitOut(t, origin, "rev-parse", "main"),
		gitOut(t, fork.Path(), "rev-parse", "foo-0.1.0^"))
}

// TestCommitCreatesMissingDirectories covers the very first publish of a
// package, where none of packages/preview/<name>/<version> exists in the fork
// yet and every level of it has to be created. Building that chain one
// directory short puts the package's files straight into
// packages/preview/<name>, which Typst Universe rejects.
func TestCommitCreatesMissingDirectories(t *testing.T) {
	t.Parallel()
	fork := cloneFork(t, setupOriginRepo(t, []string{"packages/preview/other"}))

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	commitPackage(t, fork, base, "release: foo 0.1.0")

	files := strings.Fields(gitOut(t, fork.Path(),
		"ls-tree", "-r", "--name-only", "foo-0.1.0", "packages/preview/foo/"))
	assert.Equal(t, []string{"packages/preview/foo/0.1.0/lib.typ"}, files)
}

// TestCommitReplacesVersionDirectory covers a fix-up publish that drops a file:
// the version directory is rewritten whole, so the stale file has to disappear
// from the fork rather than linger from the previous publish.
func TestCommitReplacesVersionDirectory(t *testing.T) {
	t.Parallel()
	fork := cloneFork(t, setupOriginRepo(t, []string{pkgDir}))

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	_, err = fork.Commit(publication("foo-0.1.0", []git.File{
		{Path: "lib.typ", Source: sourceFile(t, "lib.typ", "content")},
		{Path: "stale.typ", Source: sourceFile(t, "stale.typ", "gone next time")},
	}), base)
	require.NoError(t, err)

	base, err = publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	commitPackage(t, fork, base, "fix: drop stale file")

	files := strings.Fields(gitOut(t, fork.Path(),
		"ls-tree", "-r", "--name-only", "foo-0.1.0", "packages/preview/foo/0.1.0/"))
	assert.Equal(t, []string{"packages/preview/foo/0.1.0/lib.typ"}, files)
}

// TestPushSendsBranchToFork covers the end of a publish. It is also the check
// that a commit built on a blobless, shallow clone can be packed at all: the
// push has to skip the fork's other packages by hash instead of walking into
// trees the clone never fetched.
func TestPushSendsBranchToFork(t *testing.T) {
	t.Parallel()
	origin := setupOriginRepo(t, []string{pkgDir, "packages/preview/other"})
	fork := cloneFork(t, origin)

	base, err := publish.PreparePackageBranch(testLogger(), fork, "foo-0.1.0")
	require.NoError(t, err)
	local := commitPackage(t, fork, base, "release: foo 0.1.0")

	require.NoError(t, publish.Push(testLogger(), fork, "foo-0.1.0"))
	assert.Equal(t, local.String(), gitOut(t, origin, "rev-parse", "foo-0.1.0"))
	assert.Equal(t, "content",
		gitOut(t, origin, "show", "foo-0.1.0:packages/preview/foo/0.1.0/lib.typ"))
}

func testLogger() *log.Logger {
	return log.New(io.Discard)
}

// publication describes the package foo 0.1.0 being written into a fork.
func publication(branch string, files []git.File) git.Publication {
	return git.Publication{
		Branch:  branch,
		Dir:     pkgDir + "/0.1.0",
		Files:   files,
		Message: "release: foo 0.1.0",
	}
}

// commitPackage commits a one-file version of foo 0.1.0 onto branch.
func commitPackage(t *testing.T, fork *git.Fork, base git.Base, msg string) plumbing.Hash {
	t.Helper()
	pub := publication("foo-0.1.0", []git.File{
		{Path: "lib.typ", Source: sourceFile(t, "lib.typ", "content")},
	})
	pub.Message = msg
	hash, err := fork.Commit(pub, base)
	require.NoError(t, err)
	return hash
}

// sourceFile writes a package file into a temporary directory and returns it.
func sourceFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// setupOriginRepo creates a local git repo on branch main containing a marker
// file under each of pkgDirs, standing in for a subset of typst/packages'
// packages/preview/* directories. It returns its path; pass it through cloneURL
// to clone from it.
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
	cmd := exec.CommandContext(context.Background(), "git", "-C", dir,
		"show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if cmd.Run() == nil {
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
func cloneFork(t *testing.T, origin string) *git.Fork {
	t.Helper()
	fork, err := publish.EnsureForkRepo(testLogger(), cloneURL(origin), filepath.Join(t.TempDir(), "fork"))
	require.NoError(t, err)
	runGit(t, fork.Path(), "config", "user.email", "test@test.com")
	runGit(t, fork.Path(), "config", "user.name", "test")
	return fork
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
