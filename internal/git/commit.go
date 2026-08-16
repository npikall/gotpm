package git

import (
	"errors"
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
)

// ErrNoIdentity is returned when git has no user.name and user.email to
// attribute a commit to.
var ErrNoIdentity = errors.New("git has no configured identity")

// Add stages path, which may be a directory, along with everything under it
// that changed, appeared or disappeared.
func (r *Repo) Add(path string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening worktree: %w", err)
	}
	if _, err := wt.Add(path); err != nil {
		return fmt.Errorf("staging %q: %w", path, err)
	}
	return nil
}

// Commit commits the staged changes with msg and returns the new commit.
//
// The commit is built from the index, which a sparse checkout leaves holding
// the whole tree - every path outside the checkout's scope is flagged
// SkipWorktree, present in the index and absent from disk. That is what makes
// committing one package directory out of a thousand safe: the paths that were
// never materialized are carried into the new tree untouched rather than
// recorded as deletions.
//
// The author comes from git's own configuration, the user's identity as they
// already set it up. The signature does not: go-git signs only when handed a
// key, and refuses outright to commit when commit.gpgSign is set without one,
// so signing is turned off in this repository's own config first. That reaches
// no further than the clone gotpm assembles submissions in.
func (r *Repo) Commit(msg string) (string, error) {
	if err := r.disableSigning(); err != nil {
		return "", err
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("opening worktree: %w", err)
	}

	hash, err := wt.Commit(msg, &gogit.CommitOptions{})
	if errors.Is(err, gogit.ErrMissingAuthor) {
		return "", fmt.Errorf(
			"%w\nSet one:\n  git config --global user.name  \"Your Name\""+
				"\n  git config --global user.email you@example.com",
			ErrNoIdentity,
		)
	}
	if err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}
	return hash.String(), nil
}

// disableSigning records commit.gpgsign=false in this repository's own config,
// where it overrides whatever the user set globally, for this repository and
// nothing else on the machine.
func (r *Repo) disableSigning() error {
	cfg, err := r.repo.Config()
	if err != nil {
		return fmt.Errorf("reading repository config: %w", err)
	}
	if cfg.Commit.GpgSign == config.OptBoolFalse {
		return nil
	}
	cfg.Commit.GpgSign = config.NewOptBool(false)
	if err := r.repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("turning off commit signing: %w", err)
	}
	return nil
}
