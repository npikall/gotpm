package git

import (
	"fmt"
	"io"

	gogit "github.com/go-git/go-git/v6"
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

	return repo.CheckoutRevision(rev)
}

// CloneWithoutCheckout clones a repository with all of its tags but leaves the
// worktree empty, so the caller can look at what was fetched before deciding
// which revision it wants. The caller owns the repository and must close it.
func CloneWithoutCheckout(remote, dest string) (*Repo, error) {
	repo, err := gogit.PlainClone(dest, &gogit.CloneOptions{
		URL:        remote,
		Progress:   io.Discard,
		Tags:       gogit.AllTags,
		NoCheckout: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning %q into %q: %w", remote, dest, err)
	}
	return &Repo{repo: repo}, nil
}

// FetchAll updates every branch and tag from origin.
func (r *Repo) FetchAll() error {
	if err := r.repo.Fetch(&gogit.FetchOptions{Tags: gogit.AllTags, Progress: io.Discard}); err != nil {
		return fmt.Errorf("fetching: %w", err)
	}
	return nil
}

// ResolveHash returns the commit a revision points at.
func (r *Repo) ResolveHash(revision string) (string, error) {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return "", fmt.Errorf("could not resolve revision %q: %w", revision, err)
	}
	return hash.String(), nil
}

// Tags lists the tag names of the repository.
func (r *Repo) Tags() ([]string, error) {
	iter, err := r.repo.Tags()
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
func (r *Repo) CheckoutRevision(revision string) error {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return fmt.Errorf("could not resolve revision %q: %w", revision, err)
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening worktree: %w", err)
	}

	if err := wt.Checkout(&gogit.CheckoutOptions{Hash: *hash}); err != nil {
		return fmt.Errorf("checking out %q: %w", revision, err)
	}
	return nil
}
