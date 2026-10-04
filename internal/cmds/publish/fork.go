package publish

import (
	"fmt"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// EnsureForkRepo clones forkURL into forkPath if it isn't already a clone, and
// brings an existing clone's origin/main up to date.
func EnsureForkRepo(logger *log.Logger, forkURL, forkPath string) error {
	if paths.IsDir(filepath.Join(forkPath, ".git")) {
		reused, err := reuseFork(logger, forkURL, forkPath)
		if err != nil || reused {
			return err
		}
	}
	return cloneFork(logger, forkURL, forkPath)
}

// reuseFork fetches into the clone at forkPath. It reports false when the clone
// was incomplete and has been removed, so it can be cloned afresh.
func reuseFork(logger *log.Logger, forkURL, forkPath string) (bool, error) {
	if err := verifyOrigin(forkURL, forkPath); err != nil {
		return false, err
	}
	if gitcli.HasMain(forkPath) {
		return true, fetchFork(logger, forkPath)
	}
	logger.Debug("fork clone is incomplete (no origin/main), re-cloning", "path", forkPath)
	return false, wrap(paths.Remove(forkPath), "removing incomplete fork clone at %q", forkPath)
}

func cloneFork(logger *log.Logger, forkURL, forkPath string) error {
	if err := paths.EnsureDir(filepath.Dir(forkPath)); err != nil {
		return err
	}
	logger.Debug("no local fork clone found, cloning", "url", forkURL, "path", forkPath)
	err := ui.Spin("Cloning fork (first publish, this can take a while)...", func() error {
		return gitcli.Clone(forkURL, forkPath)
	})
	if err != nil {
		return fmt.Errorf("cloning fork %q: %w", forkURL, err)
	}
	logger.Debug("cloned fork")
	return nil
}

func fetchFork(logger *log.Logger, forkPath string) error {
	logger.Debug("fetching fork", "path", forkPath)
	err := ui.Spin("Fetching fork...", func() error { return gitcli.Fetch(forkPath) })
	if err != nil {
		return fmt.Errorf("fetching fork at %q: %w", forkPath, err)
	}
	logger.Debug("fetched fork")
	return nil
}

// CheckoutPackageBranch checks out branchName, scoped to pkgDir via sparse
// checkout, and makes it track origin/branchName. It reports whether the
// branch already existed - locally or on the fork - before this call.
func CheckoutPackageBranch(logger *log.Logger, forkPath, branchName, pkgDir string) (bool, error) {
	branch := inspectBranch(logger, forkPath, branchName)
	err := ui.Spin("Checking out package branch...", func() error {
		return branch.checkout(logger, pkgDir)
	})
	if err != nil {
		return false, err
	}

	if err := gitcli.SetUpstream(forkPath, branchName); err != nil {
		logger.Warn("could not set branch upstream", "branch", branchName, "err", err)
	}

	logger.Debug("checked out branch", "branch", branchName)
	return branch.existed(), nil
}

// packageBranch is a package's branch in the fork clone, and where it exists.
type packageBranch struct {
	forkPath string
	name     string
	local    bool
	onFork   bool
}

func inspectBranch(logger *log.Logger, forkPath, branchName string) packageBranch {
	onFork := gitcli.FetchBranch(forkPath, branchName) == nil
	if !onFork {
		logger.Debug("branch not on fork yet", "branch", branchName)
	}
	local := gitcli.BranchExists(forkPath, branchName)
	logger.Debug("resolved package branch", "branch", branchName, "local", local, "fork", onFork)
	return packageBranch{forkPath: forkPath, name: branchName, local: local, onFork: onFork}
}

func (b packageBranch) existed() bool {
	return b.local || b.onFork
}

func (b packageBranch) checkout(logger *log.Logger, pkgDir string) error {
	if err := gitcli.SparseCheckoutSet(b.forkPath, pkgDir); err != nil {
		return fmt.Errorf("setting sparse-checkout scope %q: %w", pkgDir, err)
	}
	if !b.local {
		return b.create(logger)
	}
	return b.update(logger)
}

// create makes the branch from the fork's copy of it, or from main when the
// fork has none.
func (b packageBranch) create(logger *log.Logger) error {
	base := "origin/main"
	if b.onFork {
		base = "origin/" + b.name
	}
	logger.Debug("creating branch", "branch", b.name, "base", base)
	return wrap(gitcli.CheckoutNewBranch(b.forkPath, b.name, base), "checking out %q", b.name)
}

// update switches to the existing branch and brings it level with the fork.
func (b packageBranch) update(logger *log.Logger) error {
	if err := gitcli.CheckoutBranch(b.forkPath, b.name); err != nil {
		return fmt.Errorf("checking out %q: %w", b.name, err)
	}
	if !b.onFork {
		return nil
	}
	return b.catchUp(logger)
}

// catchUp fast-forwards the branch onto the fork, and resets it there when the
// two have diverged (ADR 0005).
func (b packageBranch) catchUp(logger *log.Logger) error {
	logger.Debug("fast-forwarding onto fork branch", "branch", b.name)
	ffErr := gitcli.MergeFFOnly(b.forkPath, b.name)
	if ffErr == nil {
		return nil
	}
	if !gitcli.Diverged(b.forkPath, b.name) {
		return fmt.Errorf("fast-forwarding %q onto the fork: %w", b.name, ffErr)
	}

	logger.Warn("branch diverged from the fork, resetting onto it", "branch", b.name, "err", ffErr)
	return wrap(gitcli.ResetHard(b.forkPath, "origin/"+b.name), "resetting %q onto the fork", b.name)
}

func commitFork(logger *log.Logger, forkPath, relDestDir, msg string) error {
	logger.Debug("staging package files", "path", relDestDir)
	if err := gitcli.Add(forkPath, relDestDir); err != nil {
		return fmt.Errorf("staging package files: %w", err)
	}
	logger.Debug("committing to fork")
	if err := gitcli.Commit(forkPath, msg); err != nil {
		return fmt.Errorf("committing to fork: %w", err)
	}
	logger.Debug("committed to fork")
	return nil
}

// Push sends a branch to the fork's origin remote.
func Push(logger *log.Logger, forkPath, branchName string) error {
	if err := gitcli.Push(forkPath, branchName); err != nil {
		return err
	}
	logger.Debug("pushed branch to origin", "branch", branchName)
	return nil
}

// wrap annotates err with what was being done, and stays nil when err is.
func wrap(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(a, err)...) //nolint: err113 // err is the wrapped static error
}
