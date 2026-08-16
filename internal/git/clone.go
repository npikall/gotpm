package git

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// CloneRepo clones remote into dest and checks out rev.
//
// This is the plain clone, for a package's own repository: they are small
// enough that there is nothing to gain by fetching less of them, and a caller
// wants the tags, because a version is what it resolves a revision from.
// SparseClone is the other one, for a repository too large to treat this way.
func CloneRepo(remote, dest, rev string) error {
	repo, err := CloneWithoutCheckout(remote, dest)
	if err != nil {
		return err
	}
	defer repo.Close() //nolint: errcheck

	return CheckoutRevision(repo, rev)
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

// Open opens the repository at dir. The caller owns it and must close it.
func Open(dir string) (*git.Repository, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("opening repository %q: %w", dir, err)
	}
	return repo, nil
}

// FetchAll updates every branch and tag of a repository's origin.
func FetchAll(repo *git.Repository) error {
	if err := repo.Fetch(&git.FetchOptions{Tags: git.AllTags, Progress: io.Discard}); err != nil {
		return fmt.Errorf("fetching: %w", err)
	}
	return nil
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

// CheckoutRevision materializes the worktree at revision.
func CheckoutRevision(repo *git.Repository, revision string) error {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return fmt.Errorf("could not resolve revision %q: %w", revision, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening worktree: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Hash: *hash}); err != nil {
		return fmt.Errorf("checking out %q: %w", revision, err)
	}
	return nil
}
