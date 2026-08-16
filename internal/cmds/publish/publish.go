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
	git "github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkgfiles"
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
	fork, err := resolveTarget(logger)
	if err != nil {
		return err
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	branchName, m, err := commitToFork(logger, sourceDir, fork, opts.Message)
	if err != nil {
		return err
	}

	if opts.Local {
		ui.Infof("push it when you are ready:\n%s", pushCommand(fork.path, branchName))
		return nil
	}
	return pushAndSuggestPR(logger, fork, branchName, m)
}

// pushCommand returns the command that pushes branchName from forkPath
func pushCommand(forkPath, branchName string) string {
	if gitcli.TracksOwnBranch(forkPath, branchName) {
		return fmt.Sprintf("cd %s && git push", forkPath)
	}
	return fmt.Sprintf("git -C %s push origin %s", forkPath, branchName)
}

// resolveTarget loads the fork configuration, erroring if fork.url is unset,
// and resolving fork.path to its default when unset.
func resolveTarget(logger *log.Logger) (*fork, error) {
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
	forkPath, err := ResolveForkPath(logger, cfg, forkURL)
	if err != nil {
		return &fork{}, err
	}
	logger.Debug("resolved fork", "url", forkURL, "path", forkPath)
	return &fork{url: forkURL, path: forkPath}, nil
}

// commitToFork loads the package manifest, checks out its branch in the fork
// clone, copies the package files in, and commits them.
func commitToFork(
	logger *log.Logger, sourceDir string, fork *fork, msg string,
) (string, *manifest.Manifest, error) {
	// Publishing copies the package root, which is the directory holding
	// typst.toml rather than the directory the command was run from.
	manifestFile, err := manifest.FindFile(sourceDir)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	sourceDir = filepath.Dir(manifestFile)
	logger.Debug("found package", "name", m.Package.Name, "version", m.Package.Version, "root", sourceDir)

	pkgDir := path.Join("packages", previewNamespace, m.Package.Name)
	branchName := m.Package.Name + "-" + m.Package.Version

	if err := EnsureForkRepo(logger, fork.url, fork.path); err != nil {
		return "", nil, err
	}
	branchExisted, err := CheckoutPackageBranch(logger, fork.path, branchName, pkgDir)
	if err != nil {
		return "", nil, err
	}

	relDestDir := filepath.Join(filepath.FromSlash(pkgDir), m.Package.Version)
	destDir := filepath.Join(fork.path, relDestDir)
	if err := paths.Remove(destDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", nil, fmt.Errorf("clearing previous version directory: %w", err)
	}
	logger.Debug("copying package files", "src", sourceDir, "dest", destDir)
	if err := pkgfiles.CopyTree(sourceDir, destDir); err != nil {
		return "", nil, err
	}
	logger.Debug("copied package files")

	if msg == "" {
		msg = commitMessage(sourceDir, m, branchExisted)
	}
	if err := commitFork(logger, fork.path, relDestDir, msg); err != nil {
		return "", nil, err
	}
	ui.Infof("committed %q on branch %s", msg, branchName)
	return branchName, m, nil
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

// pushAndSuggestPR pushes branchName to the fork's origin remote. On failure
// it returns an error containing the equivalent manual git push command. On
// success it prints a ready-to-run `gh pr create` suggestion; gotpm never
// talks to GitHub itself.
func pushAndSuggestPR(
	logger *log.Logger, fork *fork, branchName string, m *manifest.Manifest,
) error {
	logger.Debug("pushing branch to origin", "branch", branchName)
	if err := Push(logger, fork.path, branchName); err != nil {
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
