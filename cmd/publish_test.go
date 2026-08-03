package cmd_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	. "github.com/npikall/gotpm/cmd"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckoutPackageBranchSparseScope guards against the bug where checking
// out a package that has never been published before widens the sparse
// checkout to the entire packages/preview namespace (1000+ unrelated package
// directories in the real typst/packages repo) instead of staying scoped to
// just the one package being published.
func TestCheckoutPackageBranchSparseScope(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	origin := setupOriginRepo(t, []string{
		"packages/preview/existing-pkg",
		"packages/preview/other-pkg-1",
		"packages/preview/other-pkg-2",
	})

	forkPath := t.TempDir()
	require.NoError(t, gitcli.Clone(origin, forkPath))

	logger := log.New(io.Discard)
	branchExisted, err := CheckoutPackageBranch(logger, forkPath, "new-pkg-0.1.0", "packages/preview/new-pkg")
	require.NoError(t, err)
	assert.False(t, branchExisted)

	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/existing-pkg"))
	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/other-pkg-1"))
	assert.NoDirExists(t, filepath.Join(forkPath, "packages/preview/other-pkg-2"))
}

// setupOriginRepo creates a local git repo on branch main containing an empty
// marker file under each of pkgDirs, standing in for a subset of
// typst/packages' packages/preview/* directories.
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
}
