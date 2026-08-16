// Package git is the only place gotpm talks to a git repository. It offers two
// clones: CloneRepo, for a package's own repository, and SparseClone, for one
// too large to fetch whole - the Typst Universe package repository, of which a
// publish needs a single package directory out of a thousand.
package git

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/storage"
)

// ErrNoOrigin is returned for a repository that has no origin remote to fetch
// the objects of a sparse checkout from.
var ErrNoOrigin = errors.New("repository has no origin remote")

// SparseClone clones url into dst without checking anything out, leaving it to
// SparseCheckout to decide which part of the tree is materialized. The caller
// owns the repository and must close it.
//
// It is separate from SparseCheckout because a clone happens once while its
// scope changes repeatedly: gotpm keeps one clone of a fork and publishes a
// different package, into a different directory, from it every time.
func SparseClone(dst, url string) (*git.Repository, error) {
	repo, err := git.PlainClone(dst, sparseCloneOptions(url))
	if err != nil {
		return nil, fmt.Errorf("cloning %q into %q: %w", url, dst, err)
	}
	return repo, nil
}

// SparseCheckout points repo's worktree at branch and scopes it to path,
// leaving every other path tracked but absent from disk. A path the branch
// does not have yet is not an error - publishing a package for the first time
// scopes the worktree to a directory that only the resulting commit creates.
func SparseCheckout(repo *git.Repository, branch, path string) error {
	ref := plumbing.NewBranchReferenceName(branch)
	commit, err := commitOf(repo, ref)
	if err != nil {
		return err
	}

	if err := fetchPath(repo, commit, path); err != nil {
		return err
	}

	// Reset moves whatever HEAD points at, so HEAD has to name the branch
	// before the worktree is scoped. Checkout would do both at once but has no
	// way to skip its check that the sparse directory already exists.
	symbolic := plumbing.NewSymbolicReference(plumbing.HEAD, ref)
	if err := repo.Storer.SetReference(symbolic); err != nil {
		return fmt.Errorf("pointing HEAD at %q: %w", branch, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening worktree: %w", err)
	}
	err = wt.Reset(&git.ResetOptions{
		Commit:                  commit.Hash,
		Mode:                    git.HardReset,
		SparseDirs:              []string{path},
		SkipSparseDirValidation: true,
	})
	if err != nil {
		return fmt.Errorf("scoping worktree to %q: %w", path, err)
	}
	return nil
}

// fetchPath downloads the objects of the subtree at path, which the blob
// filter of the clone left behind. It does nothing when the commit has no such
// path.
func fetchPath(repo *git.Repository, commit *object.Commit, path string) error {
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("reading tree of %s: %w", commit.Hash, err)
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		if errors.Is(err, object.ErrEntryNotFound) || errors.Is(err, object.ErrDirectoryNotFound) {
			return nil
		}
		return fmt.Errorf("looking up %q: %w", path, err)
	}

	url, err := originURL(repo)
	if err != nil {
		return err
	}
	if err := fetchObjects(context.Background(), repo.Storer, url, []plumbing.Hash{entry.Hash}); err != nil {
		return fmt.Errorf("fetching objects of %q: %w", path, err)
	}
	return nil
}

// commitOf resolves the commit a reference points at.
func commitOf(repo *git.Repository, ref plumbing.ReferenceName) (*object.Commit, error) {
	resolved, err := repo.Reference(ref, true)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", ref.Short(), err)
	}
	commit, err := repo.CommitObject(resolved.Hash())
	if err != nil {
		return nil, fmt.Errorf("reading commit %s: %w", resolved.Hash(), err)
	}
	return commit, nil
}

// originURL returns the first URL of the origin remote.
func originURL(repo *git.Repository) (string, error) {
	remote, err := repo.Remote(git.DefaultRemoteName)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoOrigin, err)
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", ErrNoOrigin
	}
	return urls[0], nil
}

func fetchObjects(ctx context.Context, st storage.Storer, rawURL string, wants []plumbing.Hash) error {
	url, err := transport.ParseURL(rawURL)
	if err != nil {
		return err //nolint: wrapcheck
	}

	cl := client.New()
	session, err := cl.Handshake(ctx, &transport.Request{
		URL:     url,
		Command: transport.UploadPackService,
	})
	if err != nil {
		return err //nolint: wrapcheck
	}
	defer func() { _ = session.Close() }()

	return session.Fetch(ctx, st, &transport.FetchRequest{ //nolint: wrapcheck
		Wants:    wants,
		Progress: io.Discard,
	})
}

// sparseCloneOptions describes a blobless, single-branch, checkout-less clone.
// The blob filter is what keeps a clone of the Typst Universe package
// repository small - its size is overwhelmingly package contents, and a
// sparsely checked out clone needs the contents of one package. The history is
// fetched in full: it is comparatively cheap, and a shallow clone would put
// every later fetch, merge and push on go-git's weakest path.
func sparseCloneOptions(url string) *git.CloneOptions {
	return &git.CloneOptions{
		URL:          url,
		NoCheckout:   true,
		SingleBranch: true,
		Filter:       packp.FilterBlobNone(),
		Progress:     io.Discard,
	}
}
