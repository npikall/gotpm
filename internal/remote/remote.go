package remote

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

var ErrParseRepoName = errors.New("could not parse repository name")

func CloneRepo(remote, dest, rev string) error {
	repo, err := git.PlainClone(dest, &git.CloneOptions{URL: remote, Progress: os.Stderr, Tags: git.AllTags})
	if err != nil {
		return fmt.Errorf("cloning %q into %q: %w", remote, dest, err)
	}
	defer repo.Close() //nolint: errcheck

	if err := CheckoutRevision(repo, rev); err != nil {
		return err
	}
	return nil
}

func CheckoutRevision(repo *git.Repository, revision string) error {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return err //nolint: wrapcheck
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err //nolint: wrapcheck
	}

	err = wt.Checkout(&git.CheckoutOptions{Hash: *hash})
	if err != nil {
		return err //nolint: wrapcheck
	}
	return nil
}

func RepoNameFromURL(remoteURL string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		return path.Base(u.Path), nil
	}
	if _, after, found := strings.Cut(trimmed, ":"); found {
		return path.Base(after), nil
	}
	return "", ErrParseRepoName
}

// OwnerFromURL returns the owner/organisation segment of a repository URL,
// e.g. "npikall" for "https://github.com/npikall/packages".
func OwnerFromURL(remoteURL string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		return path.Base(path.Dir(u.Path)), nil
	}
	if _, after, found := strings.Cut(trimmed, ":"); found {
		return path.Base(path.Dir(after)), nil
	}
	return "", ErrParseRepoName
}

func DefaultHTTPCloneURL(path string) string {
	if hasScheme(path) {
		return path
	}
	return "https://" + strings.TrimSuffix(path, ".git")
}

func hasScheme(path string) bool {
	return strings.Contains(path, "://")
}
