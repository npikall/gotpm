package git

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// ErrBranchDiverged reports a package branch that has moved on both locally and
// on the fork. Reconciling it means either discarding a local commit - which
// after `gotpm publish --local` is work deliberately not pushed yet - or
// merging, so it is left to the user.
var ErrBranchDiverged = errors.New("local branch and fork branch have diverged")

// Base describes the commit the next publish of a package should build on.
type Base struct {
	// Commit is the parent for the new commit. It is the zero hash when the
	// package branch is being created from nothing, which cannot happen while
	// origin/main resolves.
	Commit plumbing.Hash
	// Existed reports whether the branch was already published, locally or on
	// the fork, before this call.
	Existed bool
}

// ResolveBase decides which commit a new commit on branch belongs on top of,
// after bringing the fork's copy of the branch up to date.
//
// A branch present in only one place is taken from there. A branch present in
// both is fast-forwarded onto the fork's tip when the fork is ahead, and left
// on the local tip when the local clone is ahead - the unpushed-commit case.
// Anything else has diverged.
func (f *Fork) ResolveBase(branch string) (Base, error) {
	onFork, err := f.FetchBranch(branch)
	if err != nil {
		return Base{}, err
	}

	local, hasLocal := f.resolve(localRef(branch))
	remote, hasRemote := f.resolve(remoteRef(branch))
	// A fetch that reported the branch reaches here without the ref only if the
	// refspec did not land it; treat the ref itself as the authority.
	onFork = onFork && hasRemote

	switch {
	case hasLocal && onFork:
		return f.reconcile(branch, local, remote)
	case hasLocal:
		return Base{Commit: local, Existed: true}, nil
	case onFork:
		return Base{Commit: remote, Existed: true}, nil
	}

	main, ok := f.resolve(remoteRef(MainBranch))
	if !ok {
		return Base{}, fmt.Errorf("%w: %s", ErrNoMainBranch, f.path)
	}
	return Base{Commit: main}, nil
}

// reconcile chooses between the two tips of a branch that exists both locally
// and on the fork. Whichever contains the other is the newer one; neither
// containing the other means they have diverged.
func (f *Fork) reconcile(branch string, local, remote plumbing.Hash) (Base, error) {
	switch {
	// Equal, or the local clone is ahead - a `publish --local` never pushed.
	case local == remote, f.contains(local, remote):
		return Base{Commit: local, Existed: true}, nil
	case f.contains(remote, local):
		return Base{Commit: remote, Existed: true}, nil
	default:
		return Base{}, fmt.Errorf(
			"%w: %s\nInspect and reconcile them: git -C %s log --oneline %s..%s",
			ErrBranchDiverged, branch, f.path, localRef(branch), remoteRef(branch),
		)
	}
}

// contains reports whether want is an ancestor of tip.
//
// The walk stops at any commit the clone does not have rather than failing, as
// a depth-1 clone is missing every commit below its shallow boundary.
// go-git's Commit.IsAncestor is unusable here for exactly that reason: it
// surfaces the missing parent as an error instead of an answer, so a genuine
// "no" is indistinguishable from a truncated history. Stopping early can only
// turn a "yes" into a "no", which callers treat as divergence and hand to the
// user - the same safe direction as git's own --ff-only refusal.
func (f *Fork) contains(tip, want plumbing.Hash) bool {
	seen := make(map[plumbing.Hash]struct{})
	queue := []plumbing.Hash{tip}
	for len(queue) > 0 {
		h := queue[0]
		queue = queue[1:]
		if h == want {
			return true
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}

		commit, err := object.GetCommit(f.repo.Storer, h)
		if err != nil {
			continue
		}
		queue = append(queue, commit.ParentHashes...)
	}
	return false
}

// SetBranch points branch at commit.
func (f *Fork) SetBranch(branch string, commit plumbing.Hash) error {
	ref := plumbing.NewHashReference(localRef(branch), commit)
	if err := f.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("updating branch %q: %w", branch, err)
	}
	return nil
}

// SetUpstream makes branch track origin/<branch>.
//
// Without this a manual `git push` in the fork directory after `gotpm publish
// --local` offers to push onto the fork's main.
func (f *Fork) SetUpstream(branch string) error {
	cfg, err := f.repo.Config()
	if err != nil {
		return fmt.Errorf("reading clone config: %w", err)
	}
	if cfg.Branches == nil {
		cfg.Branches = map[string]*config.Branch{}
	}
	cfg.Branches[branch] = &config.Branch{
		Name:   branch,
		Remote: RemoteName,
		Merge:  localRef(branch),
	}
	if err := f.repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("setting upstream for %q: %w", branch, err)
	}
	return nil
}

// TracksOwnBranch reports whether branch tracks origin/<branch> rather than
// origin/main or nothing at all.
func (f *Fork) TracksOwnBranch(branch string) bool {
	cfg, err := f.repo.Config()
	if err != nil {
		return false
	}
	tracked, ok := cfg.Branches[branch]
	return ok && tracked.Merge == localRef(branch)
}
