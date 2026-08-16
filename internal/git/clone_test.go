package git_test

import (
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/npikall/gotpm/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneRepo(t *testing.T) { //nolint: paralleltest
	repo := setupTestRepo(t)
	dest := t.TempDir()

	gotErr := git.CloneRepo(repo, dest, "HEAD")
	require.NoError(t, gotErr)
	assert.FileExists(t, filepath.Join(dest, "README.md"))
	assert.DirExists(t, filepath.Join(dest, ".git"))
}

// setupTestRepo sets up a repo for testing purposes. It does NOT work in parallel tests
func setupTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	writeFile(t, dir, "README.md", "hello")

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)
	return dir
}
