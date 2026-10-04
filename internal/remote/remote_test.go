package remote_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestCloneWithoutCheckout_ReportsAMissingRepositoryBriefly(t *testing.T) {
	t.Parallel()
	page := "<!DOCTYPE html><html>" + strings.Repeat("<div>not found</div>", 10_000) + "</html>"
	tests := []struct {
		name   string
		status int
	}{
		{"not found", http.StatusNotFound},
		{"unauthorized, as for a missing or private repository", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(page))
			}))
			t.Cleanup(server.Close)
			url := server.URL + "/owner/repo/sub/dir"

			_, err := CloneWithoutCheckout(url, t.TempDir())

			require.ErrorIs(t, err, ErrRepositoryNotFound)
			assert.Equal(t, `repository "`+url+`" not found or not accessible`, err.Error())
		})
	}
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

func TestOwnerFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https", "https://github.com/npikall/packages", "npikall"},
		{"https with .git", "https://github.com/npikall/packages.git", "npikall"},
		{"no scheme", "github.com/npikall/packages", "npikall"},
		{"scp-like ssh", "git@github.com:npikall/packages.git", "npikall"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := OwnerFromURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOwnerFromURL_RejectsAnEmptyURL(t *testing.T) {
	t.Parallel()

	_, err := OwnerFromURL("  ")

	require.ErrorIs(t, err, ErrParseRepoName)
}

func TestTags_ListsTagsByShortName(t *testing.T) { //nolint: paralleltest
	dir := setupTestRepo(t)
	repo, err := git.PlainOpen(dir)
	require.NoError(t, err)
	head, err := repo.Head()
	require.NoError(t, err)
	for _, tag := range []string{"v0.1.0", "v0.2.0"} {
		_, err := repo.CreateTag(tag, head.Hash(), nil)
		require.NoError(t, err)
	}

	tags, err := Tags(repo)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"v0.1.0", "v0.2.0"}, tags)
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
