package git

import (
	"errors"
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// FetchMain updates origin/main, and only origin/main. Clone leaves the remote
// single-branch, so its configured refspec covers main alone and package
// branches need FetchBranch.
func (f *Fork) FetchMain() error {
	found, err := f.fetch(MainBranch)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrNoMainBranch, f.path)
	}
	return nil
}

// FetchBranch updates the fork's copy of branch, reporting whether the fork has
// it at all. A branch the fork does not have is the ordinary first-publish
// case, not an error.
func (f *Fork) FetchBranch(branch string) (bool, error) {
	return f.fetch(branch)
}

// fetch updates origin/<branch> via an explicit refspec, reporting whether the
// remote has the branch.
func (f *Fork) fetch(branch string) (bool, error) {
	err := withDegradedTransport(func(filter packp.Filter, depth int) error {
		return f.repo.Fetch(&gogit.FetchOptions{
			RemoteName: RemoteName,
			RefSpecs:   []config.RefSpec{refSpec(branch)},
			Depth:      depth,
			Tags:       plumbing.NoTags,
			Filter:     filter,
		})
	})

	switch {
	case err == nil, errors.Is(err, gogit.NoErrAlreadyUpToDate):
		return true, nil
	case isRefAbsent(err):
		return false, nil
	default:
		return false, fmt.Errorf("fetching %q from %s: %w", branch, RemoteName, err)
	}
}

// isRefAbsent reports whether err means the remote simply does not have the
// requested ref.
//
// transport.ErrEmptyRemoteRepository is in this list because of a go-git bug
// rather than because the fork is empty: under wire protocol v2 the fetch asks
// ls-refs for one ref-prefix, and a prefix matching nothing is reported as an
// empty repository instead of as a missing ref. Protocol v0/v1, where the
// server advertises every ref and go-git filters locally, correctly returns
// ErrRemoteRefNotFound for the same situation.
func isRefAbsent(err error) bool {
	return errors.Is(err, gogit.ErrRemoteRefNotFound) ||
		errors.Is(err, plumbing.ErrReferenceNotFound) ||
		errors.Is(err, transport.ErrEmptyRemoteRepository)
}

// resolve returns the commit a reference points at, and whether it exists.
func (f *Fork) resolve(name plumbing.ReferenceName) (plumbing.Hash, bool) {
	ref, err := f.repo.Reference(name, true)
	if err != nil {
		return plumbing.ZeroHash, false
	}
	return ref.Hash(), true
}
