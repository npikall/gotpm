// Package publish implements the publish command: it commits a package into a
// fork of the typst/packages repository, ready for a pull request.
package publish

import (
	"cmp"
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

// Run publishes the package of the current working directory to the configured
// fork. A package branch that has diverged from the fork is reset onto it
// (ADR 0005).
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

func pushCommand(forkPath, branchName string) string {
	if gitcli.TracksOwnBranch(forkPath, branchName) {
		return fmt.Sprintf("cd %s && git push", forkPath)
	}
	return fmt.Sprintf("git -C %s push origin %s", forkPath, branchName)
}

func resolveTarget(logger *log.Logger) (*fork, error) {
	cfg, err := config.Load()
	if err != nil {
		return &fork{}, err
	}
	forkURL, err := configuredForkURL(cfg)
	if err != nil {
		return &fork{}, err
	}
	forkPath, err := ResolveForkPath(logger, cfg, forkURL)
	if err != nil {
		return &fork{}, err
	}
	logger.Debug("resolved fork", "url", forkURL, "path", forkPath)
	return &fork{url: forkURL, path: forkPath}, nil
}

func configuredForkURL(cfg *config.Config) (string, error) {
	forkURL, err := cfg.Get("fork.url")
	if err != nil {
		return "", err
	}
	if forkURL == "" {
		return "", fmt.Errorf("%w\nRun: `gotpm config set fork.url <repo>`", ErrMissingForkURL)
	}
	return forkURL, nil
}

func commitToFork(
	logger *log.Logger, sourceDir string, fork *fork, msg string,
) (string, *manifest.Manifest, error) {
	sourceDir, m, err := loadPackage(sourceDir)
	if err != nil {
		return "", nil, err
	}
	logger.Debug("found package", "name", m.Package.Name, "version", m.Package.Version, "root", sourceDir)

	branchName := m.Package.Name + "-" + m.Package.Version
	relDestDir, branchExisted, err := stageVersion(logger, sourceDir, fork, m, branchName)
	if err != nil {
		return "", nil, err
	}

	msg = commitMessage(msg, sourceDir, m, branchExisted)
	if err := commitFork(logger, fork.path, relDestDir, msg); err != nil {
		return "", nil, err
	}
	ui.Infof("committed %q on branch %s", msg, branchName)
	return branchName, m, nil
}

// stageVersion checks out the package's branch in the fork and puts the
// package files in its version directory, which it returns relative to the
// fork. It reports whether the branch existed before.
func stageVersion(
	logger *log.Logger, sourceDir string, fork *fork, m *manifest.Manifest, branchName string,
) (string, bool, error) {
	pkgDir := path.Join("packages", previewNamespace, m.Package.Name)
	if err := EnsureForkRepo(logger, fork.url, fork.path); err != nil {
		return "", false, err
	}
	branchExisted, err := CheckoutPackageBranch(logger, fork.path, branchName, pkgDir)
	if err != nil {
		return "", false, err
	}

	relDestDir := filepath.Join(filepath.FromSlash(pkgDir), m.Package.Version)
	return relDestDir, branchExisted, replaceVersionDir(logger, sourceDir, filepath.Join(fork.path, relDestDir))
}

// loadPackage finds the manifest of the package containing dir and returns the
// package root along with it.
func loadPackage(dir string) (string, *manifest.Manifest, error) {
	manifestFile, err := manifest.FindFile(dir)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	return filepath.Dir(manifestFile), m, nil
}

// replaceVersionDir swaps destDir's contents for the package files of
// sourceDir, so files dropped from the package do not linger in the fork.
func replaceVersionDir(logger *log.Logger, sourceDir, destDir string) error {
	if err := paths.Remove(destDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clearing previous version directory: %w", err)
	}
	logger.Debug("copying package files", "src", sourceDir, "dest", destDir)
	if err := pkgfiles.CopyTree(sourceDir, destDir); err != nil {
		return err
	}
	logger.Debug("copied package files")
	return nil
}

// commitMessage is the custom message when one was given. Otherwise it reuses
// the source repository's latest commit message for a package that was
// published before, and names the release for a new one.
func commitMessage(custom, sourceDir string, m *manifest.Manifest, branchExisted bool) string {
	if custom != "" {
		return custom
	}
	release := fmt.Sprintf("release: %s %s", m.Package.Name, m.Package.Version)
	if !branchExisted {
		return release
	}
	return cmp.Or(headCommitMessage(sourceDir), release)
}

// headCommitMessage is the message of dir's latest commit, or "" when dir is
// not in a repository with one.
func headCommitMessage(dir string) string {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return ""
	}
	head, err := repo.Head()
	if err != nil {
		return ""
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(commit.Message)
}

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
