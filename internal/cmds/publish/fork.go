package publish

import (
	"fmt"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// EnsureForkRepo clones forkURL into forkPath if it isn't already a clone.
//
// The clone is blobless (--filter=blob:none) and checkout-less: only commits
// and trees are fetched up front, and git's promisor-remote support lazily
// fetches the blobs for whatever pkgDir CheckoutPackageBranch later sparse-
// checks out, rather than the ~1000 packages/preview/* directories the fork
// actually holds. go-git cannot do this - it has no mechanism to lazily fetch
// objects excluded by a server-side filter, so any later Checkout/Reset needing
// a filtered-out blob fails outright - hence shelling out to git itself.
//
// A fork clone is never fetched after its initial clone: gotpm is the only
// writer of its own branches (fix-up publishes reuse the local branch tip
// as-is), and new branches are cut from local origin/main, which staying
// pinned to clone time is fine since publishing only ever adds a new
// packages/preview/<name>/<version> directory. Deleting forkPath and letting
// the next publish re-clone is how to pick up a manually-synced fork main.
//
// A forkPath whose clone was interrupted (e.g. killed mid-clone) leaves a
// .git directory behind with no origin/main ref ever written. Such a clone is
// detected here and wiped and redone rather than failing deep inside
// CheckoutPackageBranch with a confusing "origin/main not found".
func EnsureForkRepo(logger *log.Logger, forkURL, forkPath string) error {
	if paths.IsDir(filepath.Join(forkPath, ".git")) {
		logger.Debug("using existing fork clone as-is (not fetching)", "path", forkPath)
		if gitcli.HasMain(forkPath) {
			return nil
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

// CheckoutPackageBranch checks out branchName, scoped to pkgDir via sparse
// checkout, creating it from origin/main's tip if it does not already exist.
// It reports whether the branch already existed before this call.
//
// Unlike go-git's sparse checkout, git's own sparse-checkout has no
// requirement that pkgDir already exist in the target tree, so a brand-new
// package (never published before) scopes down to just pkgDir like any other,
// rather than widening out to the whole packages/preview namespace.
func CheckoutPackageBranch(logger *log.Logger, forkPath, branchName, pkgDir string) (bool, error) {
	branchExisted := gitcli.BranchExists(forkPath, branchName)
	logger.Debug("resolved package branch", "branch", branchName, "existed", branchExisted)

	spin := ui.Spinner(" Checking out package branch...")
	spin.Start()
	defer spin.Stop()

	if err := gitcli.SparseCheckoutSet(forkPath, pkgDir); err != nil {
		return false, fmt.Errorf("setting sparse-checkout scope %q: %w", pkgDir, err)
	}

	if branchExisted {
		if err := gitcli.CheckoutBranch(forkPath, branchName); err != nil {
			return false, fmt.Errorf("checking out %q: %w", branchName, err)
		}
	} else {
		logger.Debug("creating new branch from origin/main", "branch", branchName)
		if err := gitcli.CheckoutNewBranch(forkPath, branchName, "origin/main"); err != nil {
			return false, fmt.Errorf("checking out %q: %w", branchName, err)
		}
	}
	logger.Debug("checked out branch", "branch", branchName)
	return branchExisted, nil
}

// commitFork stages relDestDir and commits it. Staging is deliberately scoped
// to relDestDir rather than the whole worktree, keeping the commit limited to
// this version's files regardless of whatever else is present in the sparse
// checkout. Signing is left to the caller's own git config as-is: real git,
// unlike go-git, natively supports whatever signing method (gpg, ssh, ...) the
// user has configured.
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
