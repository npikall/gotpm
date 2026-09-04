package publish

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	git "github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/resolve"
)

const (
	forksDirName      = "forks"
	legacyForkDirName = "fork"
)

var (
	// ErrForkOriginMismatch is returned when the clone found at the fork path
	// was cloned from a different fork than the configured one.
	ErrForkOriginMismatch = errors.New("fork clone belongs to a different fork")
	errOriginWithoutURL   = errors.New("fork clone has an origin with no url")
)

// ResolveForkPath returns the directory the fork is cloned into: the
// configured fork.path when the user set one, and otherwise the location
// derived from forkURL.
func ResolveForkPath(logger *log.Logger, cfg *config.Config, forkURL string) (string, error) {
	configured, err := cfg.Get("fork.path")
	if err != nil {
		return "", err
	}
	if configured != "" {
		return configured, nil
	}
	forkPath, err := DefaultForkPath(forkURL)
	if err != nil {
		return "", err
	}
	migrateLegacyClone(logger, forkURL, forkPath)
	return forkPath, nil
}

// DefaultForkPath derives the fork clone's location from forkURL — host, owner,
// then repository — so publishing through two forks uses two clones (ADR 0006).
func DefaultForkPath(forkURL string) (string, error) {
	dataDir, err := paths.GotpmDataDir()
	if err != nil {
		return "", err
	}
	segments, err := remote.Segments(canonicalFork(forkURL))
	if err != nil {
		return "", fmt.Errorf("deriving the fork clone path from %q: %w", forkURL, err)
	}
	return filepath.Join(append([]string{dataDir, forksDirName}, segments...)...), nil
}

func canonicalFork(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	canonical := strings.TrimSuffix(strings.TrimSuffix(trimmed, "/"), ".git")
	if src, err := resolve.Normalize(trimmed); err == nil {
		canonical = src.Canonical
	}
	if resolve.IsLocal(canonical) {
		return canonical
	}
	return strings.ToLower(canonical)
}

func sameFork(a, b string) bool {
	return canonicalFork(a) == canonicalFork(b)
}

func migrateLegacyClone(logger *log.Logger, forkURL, forkPath string) {
	dataDir, err := paths.GotpmDataDir()
	if err != nil {
		return
	}
	legacyPath := filepath.Join(dataDir, legacyForkDirName)
	if !paths.IsDir(filepath.Join(legacyPath, ".git")) {
		return
	}
	if paths.IsDir(forkPath) {
		logger.Debug("derived fork clone already exists, leaving the legacy one",
			"legacy", legacyPath, "path", forkPath)
		return
	}

	origin, err := originURL(legacyPath)
	if err != nil {
		logger.Warn("could not read the legacy fork clone's origin", "path", legacyPath, "err", err)
		return
	}
	if !sameFork(origin, forkURL) {
		logger.Warn("the fork clone at the old location is a clone of another fork; delete it when you no longer need it",
			"path", legacyPath, "origin", origin)
		return
	}

	if err := paths.EnsureDir(filepath.Dir(forkPath)); err != nil {
		logger.Warn("could not prepare the fork clone's new location", "path", forkPath, "err", err)
		return
	}
	if err := os.Rename(legacyPath, forkPath); err != nil {
		logger.Warn("could not move the fork clone to its new location",
			"from", legacyPath, "to", forkPath, "err", err)
		return
	}
	logger.Debug("moved the fork clone to its derived location", "from", legacyPath, "to", forkPath)
}

func verifyOrigin(forkURL, forkPath string) error {
	origin, err := originURL(forkPath)
	if err != nil {
		return err
	}
	if sameFork(origin, forkURL) {
		return nil
	}
	return fmt.Errorf(
		"%w: the clone at %q was cloned from %q, but fork.url is %q\n"+
			"Remove that clone, or point fork.url or fork.path where you meant to",
		ErrForkOriginMismatch, forkPath, origin, forkURL,
	)
}

func originURL(dir string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", fmt.Errorf("opening the fork clone at %q: %w", dir, err)
	}
	defer repo.Close() //nolint: errcheck

	origin, err := repo.Remote(git.DefaultRemoteName)
	if err != nil {
		return "", fmt.Errorf("reading the origin of the fork clone at %q: %w", dir, err)
	}
	urls := origin.Config().URLs
	if len(urls) == 0 {
		return "", fmt.Errorf("%w: %q", errOriginWithoutURL, dir)
	}
	return urls[0], nil
}
