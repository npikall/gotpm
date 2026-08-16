package publish

import (
	"errors"
	"fmt"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/git"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// EnsureForkRepo clones forkURL into forkPath if it isn't already a clone, and
// brings an existing clone's origin/main up to date.
func EnsureForkRepo(logger *log.Logger, forkURL, forkPath string) error {
	if paths.IsDir(filepath.Join(forkPath, ".git")) {
		complete, err := fetchExistingFork(logger, forkPath)
		if err != nil {
			return err
		}
		if complete {
			return nil
		}
		logger.Debug("fork clone is incomplete or shallow, re-cloning", "path", forkPath)
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
	repo, err := git.SparseClone(forkPath, forkURL)
	spin.Stop()
	if err != nil {
		return fmt.Errorf("cloning fork %q: %w", forkURL, err)
	}
	defer repo.Close() //nolint: errcheck
	logger.Debug("cloned fork")
	return nil
}

// fetchExistingFork brings the clone at forkPath up to date, reporting whether
// it was a complete clone to begin with. An incomplete one - no origin/main,
// so its initial clone never finished - is left for the caller to replace.
func fetchExistingFork(logger *log.Logger, forkPath string) (bool, error) {
	repo, err := git.Open(forkPath)
	if err != nil {
		return false, err
	}
	defer repo.Close() //nolint: errcheck

	// A shallow clone is one an older gotpm made with --depth 1. Its truncated
	// history is not enough to tell a fast-forward from a divergence, so it is
	// replaced rather than worked with.
	if !repo.HasMain() || repo.IsShallow() {
		return false, nil
	}

	logger.Debug("fetching fork", "path", forkPath)
	spin := ui.Spinner(" Fetching fork...")
	spin.Start()
	err = repo.Fetch()
	spin.Stop()
	if err != nil {
		return false, fmt.Errorf("fetching fork at %q: %w", forkPath, err)
	}
	logger.Debug("fetched fork")
	return true, nil
}

// ErrForkBranchDiverged is returned when the local package branch and the
// fork's branch of the same name have both moved on independently.
var ErrForkBranchDiverged = errors.New("local branch and fork branch have diverged")

// CheckoutPackageBranch checks out branchName, scoped to pkgDir via sparse
// checkout, and makes it track origin/branchName. It reports whether the
// branch already existed - locally or on the fork - before this call.
func CheckoutPackageBranch(logger *log.Logger, forkPath, branchName, pkgDir string) (bool, error) {
	repo, err := git.Open(forkPath)
	if err != nil {
		return false, err
	}
	defer repo.Close() //nolint: errcheck

	onFork := repo.FetchBranch(branchName) == nil
	if !onFork {
		logger.Debug("branch not on fork yet", "branch", branchName)
	}
	local := repo.BranchExists(branchName)
	logger.Debug("resolved package branch", "branch", branchName, "local", local, "fork", onFork)

	spin := ui.Spinner(" Checking out package branch...")
	spin.Start()
	defer spin.Stop()

	// Where the branch sits is settled first and moves references only, so a
	// clone that holds the contents of no package yet never has to materialize
	// one; the sparse checkout that follows puts pkgDir, and nothing else, on
	// disk.
	if err := positionBranch(logger, repo, forkPath, branchName, local, onFork); err != nil {
		return false, err
	}

	if err := repo.SparseCheckout(branchName, pkgDir); err != nil {
		return false, fmt.Errorf("scoping fork clone to %q: %w", pkgDir, err)
	}

	if err := repo.SetUpstream(branchName); err != nil {
		logger.Warn("could not set branch upstream", "branch", branchName, "err", err)
	}

	logger.Debug("checked out branch", "branch", branchName)
	return local || onFork, nil
}

// positionBranch puts branchName at the commit this publish should build on:
// the fork's tip when the fork has the branch, the fork's main when nobody has
// it yet, and where it already is when only this machine has it.
func positionBranch(
	logger *log.Logger, repo *git.Repo, forkPath, branchName string, local, onFork bool,
) error {
	if !local {
		base := "origin/main"
		if onFork {
			base = "origin/" + branchName
		}
		logger.Debug("creating branch", "branch", branchName, "base", base)
		if err := repo.SetBranchTo(branchName, base); err != nil {
			return fmt.Errorf("creating %q: %w", branchName, err)
		}
		return nil
	}

	if !onFork {
		return nil
	}
	logger.Debug("fast-forwarding onto fork branch", "branch", branchName)
	if err := repo.MergeFFOnly(branchName); err != nil {
		manual := fmt.Sprintf("git -C %s log --oneline %s..origin/%s", forkPath, branchName, branchName)
		return fmt.Errorf("%w: %w\nInspect and reconcile them: %s", ErrForkBranchDiverged, err, manual)
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

// tracksOwnBranch reports whether branch tracks origin/<branch> rather than
// origin/main or nothing at all.
func tracksOwnBranch(forkPath, branch string) bool {
	repo, err := git.Open(forkPath)
	if err != nil {
		return false
	}
	defer repo.Close() //nolint: errcheck
	return repo.TracksOwnBranch(branch)
}

// Push sends a branch to the fork's origin remote.
func Push(logger *log.Logger, forkPath, branchName string) error {
	if err := gitcli.Push(forkPath, branchName); err != nil {
		return err
	}
	logger.Debug("pushed branch to origin", "branch", branchName)
	return nil
}
