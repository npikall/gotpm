package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	git "github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/spf13/cobra"
)

// previewNamespace is the only namespace Typst Universe accepts submissions
// into; it is not user-configurable.
const previewNamespace = "preview"

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a Package to the Typst Universe.",
	Long: `Publish a Typst Package to the Typst Universe.
This involves pushing your changes to a fork of the github.com/typst/packages repo
on a dedicated branch, ready for you to open a Pull Request from.

GoTPM will know where your fork lives on disc, and handle committing your
Package files to the correct location.`,
	Example: `gotpm publish
gotpm publish --local`,
	RunE: publishRunner,
}

var (
	ErrMissingForkURL = errors.New("no fork has been configured")
	ErrPushFailed     = errors.New("could not push to fork")
)

func init() {
	rootCmd.AddCommand(publishCmd)
	publishCmd.Flags().Bool("local", false, "Stop after committing to the local fork clone; do not push.")
}

func publishRunner(cmd *cobra.Command, _ []string) error {
	local := internal.Must(cmd.Flags().GetBool("local"))
	logger := internal.SetupLogger(cmd)

	forkURL, forkPath, err := resolvePublishTarget()
	if err != nil {
		return err
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	branchName, manifest, err := publishToFork(logger, sourceDir, forkURL, forkPath)
	if err != nil {
		return err
	}

	if local {
		return nil
	}
	return pushAndSuggestPR(logger, forkURL, forkPath, branchName, manifest)
}

// resolvePublishTarget loads the fork configuration, erroring if fork.url is
// unset, and resolving fork.path to its default when unset.
func resolvePublishTarget() (string, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", "", err //nolint: wrapcheck
	}
	forkURL, err := cfg.Get("fork.url")
	if err != nil {
		return "", "", err //nolint: wrapcheck
	}
	if forkURL == "" {
		return "", "", fmt.Errorf("%w\nRun: `gotpm config set fork.url <repo>`", ErrMissingForkURL)
	}
	forkPath, err := resolveForkPath(cfg)
	if err != nil {
		return "", "", err
	}
	return forkURL, forkPath, nil
}

// publishToFork loads the package manifest, checks out its branch in the
// fork clone, copies the package files in, and commits them.
func publishToFork(
	logger *log.Logger, sourceDir, forkURL, forkPath string,
) (string, internal.Manifest, error) {
	manifest, err := internal.LoadManifest(sourceDir)
	if err != nil {
		return "", internal.Manifest{}, fmt.Errorf("could not load manifest: %w", err)
	}
	logger.Debug("found package", "name", manifest.Package.Name, "version", manifest.Package.Version)

	pkgDir := path.Join("packages", previewNamespace, manifest.Package.Name)
	branchName := manifest.Package.Name + "-" + manifest.Package.Version

	if err := ensureForkRepo(logger, forkURL, forkPath); err != nil {
		return "", internal.Manifest{}, err
	}
	branchExisted, err := CheckoutPackageBranch(logger, forkPath, branchName, pkgDir)
	if err != nil {
		return "", internal.Manifest{}, err
	}

	relDestDir := filepath.Join(filepath.FromSlash(pkgDir), manifest.Package.Version)
	destDir := filepath.Join(forkPath, relDestDir)
	if err := RemoveTarget(destDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", internal.Manifest{}, fmt.Errorf("clearing previous version directory: %w", err)
	}
	logger.Debug("copying package files", "src", sourceDir, "dest", destDir)
	if err := CopyPackageFiles(sourceDir, destDir); err != nil {
		return "", internal.Manifest{}, err
	}
	logger.Debug("copied package files")

	msg := commitMessage(sourceDir, manifest, branchExisted)
	if err := commitFork(logger, forkPath, relDestDir, msg); err != nil {
		return "", internal.Manifest{}, err
	}
	internal.PrintInfof("committed %q on branch %s", msg, branchName)
	return branchName, manifest, nil
}

// resolveForkPath returns the configured fork.path, defaulting to
// $APP_DATA_DIR/fork when unset.
func resolveForkPath(cfg *config.Config) (string, error) {
	forkPath, err := cfg.Get("fork.path")
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	if forkPath != "" {
		return forkPath, nil
	}
	dataDir, err := internal.ResolveAppDataDir()
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	return filepath.Join(dataDir, "fork"), nil
}

// ensureForkRepo clones forkURL into forkPath if it isn't already a clone.
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
func ensureForkRepo(logger *log.Logger, forkURL, forkPath string) error {
	if internal.IsDir(filepath.Join(forkPath, ".git")) {
		logger.Debug("using existing fork clone as-is (not fetching)", "path", forkPath)
		if gitcli.HasMain(forkPath) {
			return nil
		}
		logger.Debug("fork clone is incomplete (no origin/main), re-cloning", "path", forkPath)
		if err := RemoveTarget(forkPath); err != nil {
			return fmt.Errorf("removing incomplete fork clone at %q: %w", forkPath, err)
		}
	}
	if err := internal.EnsureDir(filepath.Dir(forkPath)); err != nil {
		return err //nolint: wrapcheck
	}
	logger.Debug("no local fork clone found, cloning", "url", forkURL, "path", forkPath)
	spin := internal.SetupSpinner()
	spin.Suffix = " Cloning fork (first publish, this can take a while)..."
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

	spin := internal.SetupSpinner()
	spin.Suffix = " Checking out package branch..."
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

// commitMessage returns "release: name version" for a fresh branch. For a
// branch that already existed (a fix-up publish run), it reuses the source
// package repo's own HEAD commit message so fixes made there carry over to
// the fork, falling back to the release message when the source isn't a git
// repo or has no commits.
func commitMessage(sourceDir string, manifest internal.Manifest, branchExisted bool) string {
	fallback := fmt.Sprintf("release: %s %s", manifest.Package.Name, manifest.Package.Version)
	if !branchExisted {
		return fallback
	}
	repo, err := git.PlainOpenWithOptions(sourceDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return fallback
	}
	head, err := repo.Head()
	if err != nil {
		return fallback
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return fallback
	}
	if msg := strings.TrimSpace(commit.Message); msg != "" {
		return msg
	}
	return fallback
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

// pushAndSuggestPR pushes branchName to the fork's origin remote. On failure
// it returns an error containing the equivalent manual git push command. On
// success it prints a ready-to-run `gh pr create` suggestion; gotpm never
// talks to GitHub itself.
func pushAndSuggestPR(
	logger *log.Logger, forkURL, forkPath, branchName string, manifest internal.Manifest,
) error {
	logger.Debug("pushing branch to origin", "branch", branchName)
	if err := gitcli.Push(forkPath, branchName); err != nil {
		manual := fmt.Sprintf("git -C %s push origin %s", forkPath, branchName)
		return fmt.Errorf("%w: %w\nRun manually: %s", ErrPushFailed, err, manual)
	}
	logger.Debug("pushed branch to origin", "branch", branchName)

	owner, err := remote.OwnerFromURL(forkURL)
	if err != nil {
		return fmt.Errorf("could not determine fork owner from %q: %w", forkURL, err)
	}
	title := fmt.Sprintf("release: %s %s", manifest.Package.Name, manifest.Package.Version)
	ghCmd := fmt.Sprintf(
		"gh pr create --repo typst/packages --base main --head %s:%s --draft --title %q",
		owner, branchName, title,
	)
	internal.PrintInfof("pushed %s. Open a PR with:\n%s", branchName, ghCmd)
	return nil
}
