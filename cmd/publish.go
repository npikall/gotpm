package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v6"
	gitconfig "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/config"
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

	forkURL, forkPath, err := resolvePublishTarget()
	if err != nil {
		return err
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repo, branchName, manifest, err := publishToFork(cmd, sourceDir, forkURL, forkPath)
	if err != nil {
		return err
	}
	defer repo.Close()

	if local {
		return nil
	}
	return pushAndSuggestPR(repo, forkURL, forkPath, branchName, manifest)
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
	cmd *cobra.Command, sourceDir, forkURL, forkPath string,
) (*git.Repository, string, internal.Manifest, error) {
	logger := internal.SetupLogger(cmd)

	manifest, err := internal.LoadManifest(sourceDir)
	if err != nil {
		return nil, "", internal.Manifest{}, fmt.Errorf("could not load manifest: %w", err)
	}
	logger.Debug("found package", "name", manifest.Package.Name, "version", manifest.Package.Version)

	pkgDir := path.Join("packages", previewNamespace, manifest.Package.Name)
	branchName := manifest.Package.Name + "-" + manifest.Package.Version

	repo, err := ensureForkRepo(forkURL, forkPath)
	if err != nil {
		return nil, "", internal.Manifest{}, err
	}
	branchExisted, err := checkoutPackageBranch(repo, branchName, pkgDir)
	if err != nil {
		return nil, "", internal.Manifest{}, err
	}

	relDestDir := filepath.Join(filepath.FromSlash(pkgDir), manifest.Package.Version)
	destDir := filepath.Join(forkPath, relDestDir)
	if err := RemoveTarget(destDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, "", internal.Manifest{}, fmt.Errorf("clearing previous version directory: %w", err)
	}
	if err := CopyPackageFiles(sourceDir, destDir); err != nil {
		return nil, "", internal.Manifest{}, err
	}

	msg := commitMessage(sourceDir, manifest, branchExisted)
	if err := commitFork(repo, relDestDir, msg); err != nil {
		return nil, "", internal.Manifest{}, err
	}
	internal.PrintInfof("committed %q on branch %s", msg, branchName)
	return repo, branchName, manifest, nil
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

// ensureForkRepo opens an already-cloned fork and fetches it, or performs a
// fresh sparse clone if forkPath does not yet hold a repository.
func ensureForkRepo(forkURL, forkPath string) (*git.Repository, error) {
	if internal.IsDir(filepath.Join(forkPath, ".git")) {
		repo, err := git.PlainOpen(forkPath)
		if err != nil {
			return nil, fmt.Errorf("opening fork at %q: %w", forkPath, err)
		}
		err = repo.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil, fmt.Errorf("fetching fork: %w", err)
		}
		return repo, nil
	}
	if err := internal.EnsureDir(filepath.Dir(forkPath)); err != nil {
		return nil, err //nolint: wrapcheck
	}
	repo, err := remote.CloneSparseCheckout(forkURL, forkPath)
	if err != nil {
		return nil, fmt.Errorf("cloning fork %q: %w", forkURL, err)
	}
	return repo, nil
}

// checkoutPackageBranch checks out branchName, scoped to pkgDir via sparse
// checkout, creating it from origin/main's tip if it does not already exist.
// It reports whether the branch already existed before this call.
//
// Checkout's sparse-checkout validation requires the sparse directory to
// already exist in the target tree, which fails for a package that has never
// been published before (its directory doesn't exist in the fork yet). When
// that's the case, this widens the sparse scope to pkgDir's parent (the
// "preview" namespace directory, which always exists) instead. A manual
// Storer/Reset-based bypass of that validation was tried first but corrupts
// the worktree/index when switching away from a previously sparse-checked-out
// branch, since Reset's cleanup diff depends on HEAD not having moved yet.
func checkoutPackageBranch(repo *git.Repository, branchName, pkgDir string) (bool, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("opening fork worktree: %w", err)
	}
	branchRef := plumbing.NewBranchReferenceName(branchName)
	existingRef, refErr := repo.Reference(branchRef, true)
	branchExisted := refErr == nil

	opts := &git.CheckoutOptions{Branch: branchRef, Force: true}
	var baseHash plumbing.Hash
	if branchExisted {
		baseHash = existingRef.Hash()
	} else {
		mainRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
		if err != nil {
			return false, fmt.Errorf("resolving origin/main on fork: %w", err)
		}
		baseHash = mainRef.Hash()
		opts.Create = true
		opts.Hash = baseHash
	}
	opts.SparseCheckoutDirectories = []string{sparseScopeFor(repo, baseHash, pkgDir)}

	if err := wt.Checkout(opts); err != nil {
		return false, fmt.Errorf("checking out %q: %w", branchName, err)
	}
	return branchExisted, nil
}

// sparseScopeFor returns pkgDir if it already exists in hash's tree, or its
// parent directory otherwise.
func sparseScopeFor(repo *git.Repository, hash plumbing.Hash, pkgDir string) string {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return path.Dir(pkgDir)
	}
	tree, err := commit.Tree()
	if err != nil {
		return path.Dir(pkgDir)
	}
	if _, err := tree.FindEntry(pkgDir); err != nil {
		return path.Dir(pkgDir)
	}
	return pkgDir
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
// to relDestDir rather than the whole worktree (AddOptions.All): when
// checkoutPackageBranch creates a new branch, go-git's Checkout leaves stale
// files from whatever was previously sparse-checked-out sitting on disk (see
// checkoutPackageBranch's doc comment) without indexing them. An All-staged
// Add would sweep those back in; scoping to the one directory we actually
// wrote keeps the commit limited to this version's files.
func commitFork(repo *git.Repository, relDestDir, msg string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening fork worktree: %w", err)
	}
	if err := wt.AddWithOptions(&git.AddOptions{Path: relDestDir}); err != nil {
		return fmt.Errorf("staging package files: %w", err)
	}
	if _, err := wt.Commit(msg, &git.CommitOptions{}); err != nil {
		return fmt.Errorf("committing to fork: %w", err)
	}
	return nil
}

// pushAndSuggestPR pushes branchName to the fork's origin remote. On failure
// it returns an error containing the equivalent manual git push command. On
// success it prints a ready-to-run `gh pr create` suggestion; gotpm never
// talks to GitHub itself.
func pushAndSuggestPR(repo *git.Repository, forkURL, forkPath, branchName string, manifest internal.Manifest) error {
	branchRef := plumbing.NewBranchReferenceName(branchName)
	refSpec := gitconfig.RefSpec(fmt.Sprintf("%s:%s", branchRef, branchRef))
	err := repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{refSpec},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		manual := fmt.Sprintf("git -C %s push origin %s", forkPath, branchName)
		return fmt.Errorf("%w: %w\nRun manually: %s", ErrPushFailed, err, manual)
	}

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
