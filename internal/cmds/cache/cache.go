// Package cache implements the cache command: it reports and clears what
// gotpm keeps on disk between runs.
package cache

import (
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/ui"
)

const bytesPerMB = 1024 * 1024

// Options holds the resolved cache flags.
type Options struct {
	// DryRun reports what would be cleared without clearing it.
	DryRun bool
	// Forks clears the fork clones instead of the cache.
	Forks bool
	// Fonts clears the font directory instead of the cache.
	Fonts bool
}

// Clear removes the cloned remote repositories, the package index cache and
// the font index, or, with opts.Forks and opts.Fonts, the fork clones and the
// font directory. Neither is cache — a fork clone may hold an unpushed commit
// (ADR 0006), the font directory holds what projects pin — so they are never
// cleared together with it.
func Clear(opts *Options, log *log.Logger) error {
	if opts.Forks || opts.Fonts {
		return clearNonCache(opts, log)
	}
	return clearCache(opts, log)
}

func clearCache(opts *Options, log *log.Logger) error {
	remotesDir, cachePath, err := cachePaths()
	if err != nil {
		return err
	}
	log.Debug("clearing", "remotes", remotesDir, "index", cachePath)
	size, err := paths.Size(remotesDir, cachePath)
	if err != nil {
		return err
	}
	if opts.DryRun {
		ui.Warnf("dry-run, would clear %s \n remotes: %q\n index cache: %q", format(size), remotesDir, cachePath)
		return nil
	}
	return clearAll(size, remotesDir, cachePath)
}

// cachePaths are where the cloned remotes and the index cache are kept.
func cachePaths() (string, string, error) {
	remotesDir, err := remote.CacheDir()
	if err != nil {
		return "", "", err
	}
	cachePath, err := index.CachePath()
	return remotesDir, cachePath, err
}

func clearAll(size int64, remotesDir, cachePath string) error {
	if err := remote.ClearCache(); err != nil {
		return err
	}
	if err := index.ClearCache(); err != nil {
		return err
	}
	if err := fonts.ClearIndex(); err != nil {
		return err
	}
	ui.Infof("cleared %s \n remotes: %q\n index cache: %q", format(size), remotesDir, cachePath)
	return nil
}

// clearNonCache removes the fork clones and the font directory, whichever
// opts asks for.
func clearNonCache(opts *Options, log *log.Logger) error {
	if opts.Forks {
		if err := clearDir("fork clones", paths.GotpmForksDir, opts, log); err != nil {
			return err
		}
	}
	if opts.Fonts {
		return clearDir("fonts", paths.GotpmFontsDir, opts, log)
	}
	return nil
}

// clearDir removes a directory gotpm keeps under its data directory: every
// fork clone, or every installed font family. A fork.path the user configured
// is theirs and is left alone.
func clearDir(what string, locate func() (string, error), opts *Options, log *log.Logger) error {
	dir, size, err := measure(locate)
	if err != nil {
		return err
	}
	log.Debug("clearing", "what", what, "dir", dir)
	if opts.DryRun {
		ui.Warnf("dry-run, would clear %s \n %s: %q", format(size), what, dir)
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("could not remove %s %q: %w", what, dir, err)
	}
	ui.Infof("cleared %s \n %s: %q", format(size), what, dir)
	return nil
}

// measure locates a directory and sums what it holds.
func measure(locate func() (string, error)) (string, int64, error) {
	dir, err := locate()
	if err != nil {
		return "", 0, err
	}
	size, err := paths.Size(dir)
	return dir, size, err
}

func format(size int64) string {
	return fmt.Sprintf("%.1fMB", float64(size)/bytesPerMB)
}
