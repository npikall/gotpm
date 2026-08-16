package git

import (
	"context"
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

// SparseRepo represents a (large) Repo, that should be checked out sparsely
type SparseRepo struct {
	// The url of where the repo's remote
	URL string
	// Represents the path that should be checked out
	Path string
	// The Branch that should be checked out
	Branch string
}

// SparseClone clones a repository from url into dst
// and does a sparse-checkout on path
func SparseClone(dst string, sr SparseRepo) error {
	ctx := context.Background()
	branch := plumbing.NewBranchReferenceName(sr.Branch)
	opts := sparseCloneOptions(sr.URL, branch)

	// Blobless Clone, because we fetch objects later
	repo, err := git.PlainClone(dst, opts)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}
	defer func() { _ = repo.Close() }()

	tree, err := resolveTree(repo, sr.Path)
	if err != nil {
		return fmt.Errorf("resolving tree failed: %w", err)
	}

	if err := fetchObjects(ctx, repo.Storer, sr.URL, []plumbing.Hash{tree.Hash}); err != nil {
		return err
	}

	return nil
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

func resolveTree(repo *git.Repository, path string) (*object.TreeEntry, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return entry, nil
}

// sparseCloneOptions describes a blobless, single-branch, checkout-less clone.
// The blob filter is what keeps a clone of the Typst Universe package
// repository small - its size is overwhelmingly package contents, and a
// sparsely checked out clone needs the contents of one package. The history is
// fetched in full: it is comparatively cheap, and a shallow clone would put
// every later fetch, merge and push on go-git's weakest path.
func sparseCloneOptions(url string, branch plumbing.ReferenceName) *git.CloneOptions {
	return &git.CloneOptions{
		URL:           url,
		NoCheckout:    true,
		SingleBranch:  true,
		ReferenceName: branch,
		Filter:        packp.FilterBlobNone(),
		Progress:      io.Discard,
	}
}
