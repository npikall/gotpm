// Package publish implements the publish command: it commits a package into a
// fork of the typst/packages repository, ready for a pull request.
package publish

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	gogit "github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/git"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/ui"
)

// previewNamespace is the only namespace Typst Universe accepts submissions
// into; it is not user-configurable.
const previewNamespace = "preview"

var (
	ErrMissingForkURL = errors.New("no fork has been configured")
	ErrPushFailed     = errors.New("could not push to fork")
)

// Options holds the resolved publish flags.
type Options struct {
	// Local stops after committing to the fork clone, without pushing.
	Local bool
	// Custom commit message
	Message string
}

type fork struct {
	url  string
	path string
}

// Run publishes the package of the current working directory to the
// configured fork.
func Run(opts *Options, logger *log.Logger) error {
	fork, err := resolveTarget()
	if err != nil {
		return err
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	clone, branchName, m, err := commitToFork(logger, sourceDir, fork, opts.Message)
	if err != nil {
		return err
	}

	if opts.Local {
		ui.Infof("push it when you are ready:\n%s", pushCommand(clone, branchName))
		return nil
	}
	return pushAndSuggestPR(logger, clone, fork, branchName, m)
}

// pushCommand returns the command that pushes branchName from the fork clone.
func pushCommand(clone *git.Fork, branchName string) string {
	if clone.TracksOwnBranch(branchName) {
		return fmt.Sprintf("cd %s && git push", clone.Path())
	}
	return fmt.Sprintf("git -C %s push origin %s", clone.Path(), branchName)
}

// resolveTarget loads the fork configuration, erroring if fork.url is unset,
// and resolving fork.path to its default when unset.
func resolveTarget() (*fork, error) {
	cfg, err := config.Load()
	if err != nil {
		return &fork{}, err
	}
	forkURL, err := cfg.Get("fork.url")
	if err != nil {
		return &fork{}, err
	}
	if forkURL == "" {
		return &fork{}, fmt.Errorf("%w\nRun: `gotpm config set fork.url <repo>`", ErrMissingForkURL)
	}
	forkPath, err := resolveForkPath(cfg)
	if err != nil {
		return &fork{}, err
	}
	return &fork{url: forkURL, path: forkPath}, nil
}

// resolveForkPath returns the configured fork.path, defaulting to
// $APP_DATA_DIR/fork when unset.
func resolveForkPath(cfg *config.Config) (string, error) {
	forkPath, err := cfg.Get("fork.path")
	if err != nil {
		return "", err
	}
	if forkPath != "" {
		return forkPath, nil
	}
	dataDir, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "fork"), nil
}

// commitToFork loads the package manifest, resolves the base its branch builds
// on in the fork clone, and commits the package files onto it. It returns the
// clone so the caller can push from it.
func commitToFork(
	logger *log.Logger, sourceDir string, fork *fork, msg string,
) (*git.Fork, string, *manifest.Manifest, error) {
	// Publishing copies the package root, which is the directory holding
	// typst.toml rather than the directory the command was run from.
	manifestFile, err := manifest.FindFile(sourceDir)
	if err != nil {
		return nil, "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return nil, "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	sourceDir = filepath.Dir(manifestFile)
	logger.Debug("found package", "name", m.Package.Name, "version", m.Package.Version, "root", sourceDir)

	pkgDir := path.Join("packages", previewNamespace, m.Package.Name)
	branchName := m.Package.Name + "-" + m.Package.Version

	clone, err := EnsureForkRepo(logger, fork.url, fork.path)
	if err != nil {
		return nil, "", nil, err
	}
	base, err := PreparePackageBranch(logger, clone, branchName)
	if err != nil {
		return nil, "", nil, err
	}

	// The version directory is written whole, so a file dropped from the
	// package since the last publish disappears from the fork too.
	files, err := collectFiles(sourceDir)
	if err != nil {
		return nil, "", nil, err
	}
	logger.Debug("collected package files", "src", sourceDir, "count", len(files))

	if msg == "" {
		msg = commitMessage(sourceDir, m, base.Existed)
	}
	pub := git.Publication{
		Branch:  branchName,
		Dir:     path.Join(pkgDir, m.Package.Version),
		Files:   files,
		Message: msg,
	}
	if err := commitFork(logger, clone, pub, base); err != nil {
		return nil, "", nil, err
	}
	ui.Infof("committed %q on branch %s", msg, branchName)
	return clone, branchName, m, nil
}

// commitMessage returns "release: name version" for a fresh branch. For a
// branch that already existed (a fix-up publish run), it reuses the source
// package repo's own HEAD commit message so fixes made there carry over to
// the fork, falling back to the release message when the source isn't a git
// repo or has no commits.
func commitMessage(sourceDir string, m *manifest.Manifest, branchExisted bool) string {
	fallback := fmt.Sprintf("release: %s %s", m.Package.Name, m.Package.Version)
	if !branchExisted {
		return fallback
	}
	repo, err := gogit.PlainOpenWithOptions(sourceDir, &gogit.PlainOpenOptions{DetectDotGit: true})
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

// pushAndSuggestPR pushes branchName to the fork's origin remote. On failure
// it returns an error containing the equivalent manual git push command. On
// success it prints a ready-to-run `gh pr create` suggestion; gotpm never
// talks to GitHub itself.
func pushAndSuggestPR(
	logger *log.Logger, clone *git.Fork, fork *fork, branchName string, m *manifest.Manifest,
) error {
	logger.Debug("pushing branch to origin", "branch", branchName)
	if err := Push(logger, clone, branchName); err != nil {
		manual := fmt.Sprintf("git -C %s push origin %s", fork.path, branchName)
		return fmt.Errorf("%w: %w\nRun manually: %s", ErrPushFailed, err, manual)
	}

	owner, err := remote.OwnerFromURL(fork.url)
	if err != nil {
		return fmt.Errorf("could not determine fork owner from %q: %w", fork.url, err)
	}
	title := fmt.Sprintf("release: %s %s", m.Package.Name, m.Package.Version)
	ghCmd := fmt.Sprintf(
		"gh pr create --repo typst/packages --base main --head %s:%s --draft --title %q",
		owner, branchName, title,
	)
	ui.Infof("pushed %s. Open a PR with:\n%s", branchName, ghCmd)
	return nil
}
