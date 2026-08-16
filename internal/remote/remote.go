// Package remote deals with remote package repositories: parsing their URLs,
// cloning them and caching those clones on disk.
package remote

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

var ErrParseRepoName = errors.New("could not parse repository name")

func CloneRepo(remote, dest, rev string) error {
	repo, err := CloneWithoutCheckout(remote, dest)
	if err != nil {
		return err
	}
	defer repo.Close() //nolint: errcheck

	if err := CheckoutRevision(repo, rev); err != nil {
		return err
	}
	return nil
}

// CloneWithoutCheckout clones a repository with all of its tags but leaves the
// worktree empty, so the caller can look at what was fetched before deciding
// which revision it wants. The caller owns the repository and must close it.
func CloneWithoutCheckout(remote, dest string) (*git.Repository, error) {
	repo, err := git.PlainClone(dest, &git.CloneOptions{
		URL:        remote,
		Progress:   io.Discard,
		Tags:       git.AllTags,
		NoCheckout: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning %q into %q: %w", remote, dest, err)
	}
	return repo, nil
}

// ResolveHash returns the commit a revision points at.
func ResolveHash(repo *git.Repository, revision string) (string, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return "", fmt.Errorf("could not resolve revision %q: %w", revision, err)
	}
	return hash.String(), nil
}

// Tags lists the tag names of a repository.
func Tags(repo *git.Repository) ([]string, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("could not list tags: %w", err)
	}
	defer iter.Close()

	var tags []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not list tags: %w", err)
	}
	return tags, nil
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

// CanonicalURL reduces the spellings of one repository url to a single form —
// host, owner and repository, with no scheme, no credentials and no ".git"
// suffix — so that "https://github.com/me/packages" and
// "git@github.com:me/packages.git" are recognisably the same repository.
//
// A file:// url keeps its scheme: that is what marks a repository living on
// this machine, and dropping it would make a local path indistinguishable from
// a host.
func CanonicalURL(rawURL string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(rawURL), ".git")
	if strings.HasPrefix(trimmed, fileScheme) {
		return trimmed
	}
	if u, err := url.Parse(trimmed); err == nil && u.Host != "" {
		return strings.TrimSuffix(u.Host+u.Path, "/")
	}
	// The scp-like form git uses for ssh, which is not a url: the colon
	// separates the host from the path rather than naming a port.
	if host, repoPath, found := strings.Cut(trimmed, ":"); found {
		if _, afterUser, hasUser := strings.Cut(host, "@"); hasUser {
			host = afterUser
		}
		return strings.TrimSuffix(host+"/"+strings.TrimPrefix(repoPath, "/"), "/")
	}
	return strings.TrimSuffix(trimmed, "/")
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
