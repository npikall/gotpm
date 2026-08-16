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
		if err := verifyOrigin(forkURL, forkPath); err != nil {
			return err
		}
		if gitcli.HasMain(forkPath) {
			return fetchFork(logger, forkPath)
		}
		logger.Debug("fork clone is incomplete (no origin/main), re-cloning", "path", forkPath)
		if err := paths.Remove(forkPath); err != nil {
			return fmt.Errorf("removing incomplete fork clone at %q: %w", forkPath, err)
		}
	}
	if err := paths.EnsureDir(filepath.Dir(forkPath)); err != nil {
		return err
	}
	logger.Debug("no local fork clone found, cloning", "url", forkURL, "path", forkPath)
	spin := ui.Spinner(" Cloning fork (first publish, this can take a while)...")
	spin.Start()
	err := gitcli.Clone(forkURL, forkPath)
	spin.Stop()
	if err != nil {
		return fmt.Errorf("cloning fork %q: %w", forkURL, err)
	}
	logger.Debug("cloned fork")
	return nil
}

// fetchFork updates an existing clone's origin/main.
func fetchFork(logger *log.Logger, forkPath string) error {
	logger.Debug("fetching fork", "path", forkPath)
	spin := ui.Spinner(" Fetching fork...")
	spin.Start()
	err := gitcli.Fetch(forkPath)
	spin.Stop()
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
	onFork := gitcli.FetchBranch(forkPath, branchName) == nil
	if !onFork {
		logger.Debug("branch not on fork yet", "branch", branchName)
	}
	local := gitcli.BranchExists(forkPath, branchName)
	logger.Debug("resolved package branch", "branch", branchName, "local", local, "fork", onFork)

	spin := ui.Spinner(" Checking out package branch...")
	spin.Start()
	defer spin.Stop()

	if err := gitcli.SparseCheckoutSet(forkPath, pkgDir); err != nil {
		return false, fmt.Errorf("setting sparse-checkout scope %q: %w", pkgDir, err)
	}

	if err := checkoutFrom(logger, forkPath, branchName, local, onFork); err != nil {
		return false, err
	}

	if err := gitcli.SetUpstream(forkPath, branchName); err != nil {
		logger.Warn("could not set branch upstream", "branch", branchName, "err", err)
	}

	logger.Debug("checked out branch", "branch", branchName)
	return local || onFork, nil
}

// checkoutFrom checks out branchName from whichever base the branch's
// local/fork existence calls for, fast-forwarding onto the fork's tip when the
// branch exists in both places, and resetting onto it when the two diverged.
func checkoutFrom(logger *log.Logger, forkPath, branchName string, local, onFork bool) error {
	if !local {
		base := "origin/main"
		if onFork {
			base = "origin/" + branchName
		}
		logger.Debug("creating branch", "branch", branchName, "base", base)
		if err := gitcli.CheckoutNewBranch(forkPath, branchName, base); err != nil {
			return fmt.Errorf("checking out %q: %w", branchName, err)
		}
		return nil
	}

	if err := gitcli.CheckoutBranch(forkPath, branchName); err != nil {
		return fmt.Errorf("checking out %q: %w", branchName, err)
	}
	if !onFork {
		return nil
	}
	logger.Debug("fast-forwarding onto fork branch", "branch", branchName)
	ffErr := gitcli.MergeFFOnly(forkPath, branchName)
	if ffErr == nil {
		return nil
	}
	// A fast-forward can also be refused for reasons that are not divergence
	// - a busy index, a git that cannot run at all. Those are reported rather
	// than resolved by force.
	if !gitcli.Diverged(forkPath, branchName) {
		return fmt.Errorf("fast-forwarding %q onto the fork: %w", branchName, ffErr)
	}

	// The two have diverged. The fork clone is a staging area, and every
	// commit on a package branch is a copy of the package's working tree,
	// which the publish run this is part of is about to re-copy and
	// re-commit. Resetting onto the fork keeps what only exists there - an
	// edit made on GitHub after the Universe maintainers asked for one - and
	// drops only commits whose content is regenerated a moment later.
	logger.Warn("branch diverged from the fork, resetting onto it", "branch", branchName, "err", ffErr)
	if err := gitcli.ResetHard(forkPath, "origin/"+branchName); err != nil {
		return fmt.Errorf("resetting %q onto the fork: %w", branchName, err)
	}
	return nil
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
