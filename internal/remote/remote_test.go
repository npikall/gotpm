package remote_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	. "github.com/npikall/gotpm/internal/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneRepo(t *testing.T) { //nolint: paralleltest
	repo := setupTestRepo(t)
	dest := t.TempDir()

	gotErr := CloneRepo(repo, dest, "HEAD")
	require.NoError(t, gotErr)
	assert.FileExists(t, filepath.Join(dest, "README.md"))
	assert.DirExists(t, filepath.Join(dest, ".git"))
}

func TestDefaultHTTPCloneURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"has http", "http://github.com/user/repo.git", "http://github.com/user/repo.git"},
		{"has no scheme", "github.com/user/repo.git", "https://github.com/user/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DefaultHTTPCloneURL(tt.got)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCanonicalURL covers the spellings of one repository users actually type.
// Anything that reduces two of them to different strings makes gotpm treat one
// repository as two.
func TestCanonicalURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"no scheme", "github.com/me/packages", "github.com/me/packages"},
		{"https", "https://github.com/me/packages", "github.com/me/packages"},
		{"https with .git", "https://github.com/me/packages.git", "github.com/me/packages"},
		{"http", "http://github.com/me/packages", "github.com/me/packages"},
		{"trailing slash", "https://github.com/me/packages/", "github.com/me/packages"},
		{"surrounding space", "  github.com/me/packages  ", "github.com/me/packages"},
		{"scp form", "git@github.com:me/packages.git", "github.com/me/packages"},
		{"ssh scheme", "ssh://git@github.com/me/packages", "github.com/me/packages"},
		{"credentials", "https://token@github.com/me/packages", "github.com/me/packages"},
		{"another owner stays apart", "github.com/acme/packages", "github.com/acme/packages"},
		{"file url keeps its scheme", "file:///tmp/fork", "file:///tmp/fork"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CanonicalURL(tt.got))
		})
	}
}

// setupTestRepo sets up a repo for testing purposes. It does NOT work in parallel tests
func setupTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	err = writeFile(dir, "README.md", "hello")
	require.NoError(t, err)

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)
	return dir
}

func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644) //nolint: wrapcheck
}
