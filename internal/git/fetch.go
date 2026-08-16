package git

import (
	"errors"
	"fmt"
	"io"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
)

// Fetch updates the refs the clone's own refspec covers. For a clone made by
// SparseClone that is origin/main and nothing else, because it is a
// single-branch clone: a package's branch is not covered by it and needs
// FetchBranch.
func (r *Repo) Fetch() error {
	return r.fetch(&gogit.FetchOptions{})
}

// FetchBranch updates origin/<branch> through an explicit refspec, whatever
// the clone's own refspec covers. Its error is how a caller learns that the
// branch does not exist on the remote at all.
func (r *Repo) FetchBranch(branch string) error {
	spec := config.RefSpec(fmt.Sprintf(
		"+%s:%s",
		plumbing.NewBranchReferenceName(branch),
		plumbing.NewRemoteReferenceName(gogit.DefaultRemoteName, branch),
	))
	return r.fetch(&gogit.FetchOptions{RefSpecs: []config.RefSpec{spec}})
}

// FetchAll updates every branch and tag from origin.
func (r *Repo) FetchAll() error {
	return r.fetch(&gogit.FetchOptions{Tags: gogit.AllTags})
}

// fetch fills in what every fetch shares - the remote, its credentials, and
// the silence - and reports having nothing to do as success, which is what the
// git binary did by exiting zero.
func (r *Repo) fetch(opts *gogit.FetchOptions) error {
	url, err := r.originURL()
	if err != nil {
		return err
	}

	opts.RemoteName = gogit.DefaultRemoteName
	opts.Progress = io.Discard
	opts.ClientOptions = clientOptions(url)

	if err := r.repo.Fetch(opts); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetching from %q: %w", url, err)
	}
	return nil
}
