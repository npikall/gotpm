package git

import (
	"errors"
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
)

// ErrNotFastForward is returned when a branch and its remote counterpart have
// both moved on independently, so one cannot be fast-forwarded onto the other.
var ErrNotFastForward = errors.New("branches have diverged")

// SetBranchTo points branch at whatever base resolves to, creating the branch
// or moving it. It is `git checkout -B branch base` with the checkout left
// out: where a branch sits is decided here, what that puts on disk is
// SparseCheckout's business, and keeping the two apart is what stops a scoped
// clone from ever having to materialize the whole tree.
func (r *Repo) SetBranchTo(branch, base string) error {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(base))
	if err != nil {
		return fmt.Errorf("could not resolve %q: %w", base, err)
	}
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), *hash)
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("pointing %q at %q: %w", branch, base, err)
	}
	return nil
}

// MergeFFOnly fast-forwards branch onto origin/<branch>, refusing rather than
// merging when the two have diverged. Only the branch reference moves; the
// worktree is left to SparseCheckout.
//
// The ancestry is checked here rather than through go-git's Merge, which
// fast-forwards anyway when a shallow clone truncates the history before the
// answer is reached. Guessing in that direction discards whatever the local
// branch had that the fork did not - after `gotpm publish --local`, work
// deliberately not pushed yet - so an unprovable fast-forward is refused.
func (r *Repo) MergeFFOnly(branch string) error {
	local, err := r.commitOf(plumbing.NewBranchReferenceName(branch))
	if err != nil {
		return err
	}
	remoteName := plumbing.NewRemoteReferenceName(gogit.DefaultRemoteName, branch)
	remote, err := r.commitOf(remoteName)
	if err != nil {
		return err
	}

	if local.Hash == remote.Hash {
		return nil
	}

	ancestor, err := local.IsAncestor(remote)
	if err != nil {
		return fmt.Errorf("comparing %q with %q: %w", branch, remoteName.Short(), err)
	}
	if !ancestor {
		return fmt.Errorf("%w: %s and %s", ErrNotFastForward, branch, remoteName.Short())
	}

	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), remote.Hash)
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("fast-forwarding %q onto %q: %w", branch, remoteName.Short(), err)
	}
	return nil
}

// IsShallow reports whether the repository's history is truncated. gotpm does
// not make shallow clones - a clone made by an older version is replaced
// rather than worked with, because go-git resolves questions it cannot answer
// from truncated history by assuming the answer it prefers.
func (r *Repo) IsShallow() bool {
	shallows, err := r.repo.Storer.Shallow()
	return err == nil && len(shallows) > 0
}

// SetUpstream makes branch track the branch of the same name on origin.
func (r *Repo) SetUpstream(branch string) error {
	cfg, err := r.repo.Config()
	if err != nil {
		return fmt.Errorf("reading repository config: %w", err)
	}
	if cfg.Branches == nil {
		cfg.Branches = map[string]*config.Branch{}
	}
	cfg.Branches[branch] = &config.Branch{
		Name:   branch,
		Remote: gogit.DefaultRemoteName,
		Merge:  plumbing.NewBranchReferenceName(branch),
	}
	if err := r.repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("making %q track %s/%s: %w", branch, gogit.DefaultRemoteName, branch, err)
	}
	return nil
}

// setHEAD points HEAD at branch without touching the worktree.
func (r *Repo) setHEAD(branch string) error {
	ref := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branch))
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("pointing HEAD at %q: %w", branch, err)
	}
	return nil
}
