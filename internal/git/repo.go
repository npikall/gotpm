package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// Repo is an open git repository. It is the only handle gotpm has on one: the
// go-git repository behind it stays unexported, so everything that talks to
// git does so through this package and nothing else has to know how.
//
// A repository is opened once and used many times. The callers that used to
// run a git subprocess per question ask a Repo instead, and close it when they
// are done.
type Repo struct {
	repo *gogit.Repository
}

// Open opens the repository at dir. The caller owns it and must close it.
func Open(dir string) (*Repo, error) {
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("opening repository %q: %w", dir, err)
	}
	return &Repo{repo: repo}, nil
}

// Close releases the repository.
func (r *Repo) Close() error {
	return r.repo.Close() //nolint: wrapcheck
}

// HasMain reports whether origin/main resolves, i.e. whether the clone this
// repository sits in ever finished.
func (r *Repo) HasMain() bool {
	_, err := r.repo.Reference(plumbing.NewRemoteReferenceName(gogit.DefaultRemoteName, defaultBranch), true)
	return err == nil
}

// BranchExists reports whether branch exists locally.
func (r *Repo) BranchExists(branch string) bool {
	_, err := r.repo.Reference(plumbing.NewBranchReferenceName(branch), false)
	return err == nil
}

// TracksOwnBranch reports whether branch tracks the branch of the same name on
// origin, rather than origin/main or nothing at all.
func (r *Repo) TracksOwnBranch(branch string) bool {
	cfg, err := r.repo.Config()
	if err != nil {
		return false
	}
	tracked, ok := cfg.Branches[branch]
	if !ok {
		return false
	}
	return tracked.Merge == plumbing.NewBranchReferenceName(branch)
}

// defaultBranch is the branch a fork of the Typst Universe package repository
// is published against.
const defaultBranch = "main"
