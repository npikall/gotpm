package publish

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/npikall/gotpm/internal/git"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkgfiles"
	"github.com/npikall/gotpm/internal/ui"
)

// ErrForkBranchDiverged is returned when the local package branch and the
// fork's branch of the same name have both moved on independently.
var ErrForkBranchDiverged = git.ErrBranchDiverged

// EnsureForkRepo clones forkURL into forkPath if it isn't already a clone, and
// brings an existing clone's origin/main up to date.
func EnsureForkRepo(logger *log.Logger, forkURL, forkPath string) (*git.Fork, error) {
	if fork, ok := openExisting(logger, forkPath); ok {
		return fork, fetchFork(logger, fork)
	}

	if err := paths.EnsureDir(filepath.Dir(forkPath)); err != nil {
		return nil, err
	}
	logger.Debug("no local fork clone found, cloning", "url", forkURL, "path", forkPath)

	fork, err := ui.WithSpinner(
		"Cloning fork (first publish, this can take a while)...",
		func() (*git.Fork, error) { return git.Clone(forkURL, forkPath) },
	)
	if err != nil {
		return nil, err
	}
	logger.Debug("cloned fork")
	return fork, nil
}

// openExisting opens the clone at forkPath, reporting whether there is a usable
// one there. A directory left behind by an interrupted clone has no resolvable
// origin/main; it is removed so the caller clones afresh.
func openExisting(logger *log.Logger, forkPath string) (*git.Fork, bool) {
	if !paths.IsDir(filepath.Join(forkPath, ".git")) {
		return nil, false
	}
	fork, err := git.Open(forkPath)
	if err == nil && fork.HasMain() {
		return fork, true
	}
	logger.Debug("fork clone is incomplete, re-cloning", "path", forkPath, "err", err)
	if err := paths.Remove(forkPath); err != nil {
		logger.Warn("could not remove incomplete fork clone", "path", forkPath, "err", err)
	}
	return nil, false
}

// fetchFork updates an existing clone's origin/main. A fork cloned once and
// reused for months would otherwise cut new branches from whatever main was at
// clone time.
func fetchFork(logger *log.Logger, fork *git.Fork) error {
	logger.Debug("fetching fork", "path", fork.Path())
	if err := ui.Spin("Fetching fork...", fork.FetchMain); err != nil {
		return fmt.Errorf("fetching fork at %q: %w", fork.Path(), err)
	}
	logger.Debug("fetched fork")
	return nil
}

// PreparePackageBranch resolves the commit branchName's next commit belongs on
// top of, and makes the branch track origin/branchName. It reports whether the
// branch already existed - locally or on the fork - before this call.
func PreparePackageBranch(logger *log.Logger, fork *git.Fork, branchName string) (git.Base, error) {
	base, err := ui.WithSpinner("Resolving package branch...", func() (git.Base, error) {
		return fork.ResolveBase(branchName)
	})
	if err != nil {
		return git.Base{}, err
	}
	logger.Debug("resolved package branch",
		"branch", branchName, "base", base.Commit, "existed", base.Existed)

	if err := fork.SetUpstream(branchName); err != nil {
		logger.Warn("could not set branch upstream", "branch", branchName, "err", err)
	}
	return base, nil
}

// commitFork commits the package files into the fork, without checking
// anything out.
func commitFork(logger *log.Logger, fork *git.Fork, pub git.Publication, base git.Base) error {
	logger.Debug("committing to fork", "dir", pub.Dir, "files", len(pub.Files))
	hash, err := ui.WithSpinner("Committing package...", func() (plumbing.Hash, error) {
		return fork.Commit(pub, base)
	})
	if err != nil {
		return fmt.Errorf("committing to fork: %w", err)
	}
	logger.Debug("committed to fork", "commit", hash)
	return nil
}

// Push sends a branch to the fork's origin remote.
func Push(logger *log.Logger, fork *git.Fork, branchName string) error {
	if err := ui.Spin("Pushing branch...", func() error { return fork.Push(branchName) }); err != nil {
		return err
	}
	logger.Debug("pushed branch to origin", "branch", branchName)
	return nil
}

// collectFiles gathers the package's files, as paths relative to the package
// root, ready to be written into the fork as blobs. It applies the same ignore
// rules a copy into a worktree would.
func collectFiles(sourceDir string) ([]git.File, error) {
	jobs, err := pkgfiles.Collect(sourceDir, "", pkgfiles.Matcher(sourceDir))
	if err != nil {
		return nil, err
	}

	files := make([]git.File, 0, len(jobs))
	for _, job := range jobs {
		info, err := os.Stat(job.Src)
		if err != nil {
			return nil, fmt.Errorf("reading file info %q: %w", job.Src, err)
		}
		files = append(files, git.File{
			Path:       filepath.ToSlash(job.Dst),
			Source:     job.Src,
			Executable: isExecutable(info.Mode()),
		})
	}
	return files, nil
}

// isExecutable reports whether a file's mode should be recorded as git's
// executable mode, which git derives from the owner-execute bit alone.
func isExecutable(mode fs.FileMode) bool {
	return mode&0o100 != 0
}
