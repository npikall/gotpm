// Package cache implements the cache command: it reports and clears what
// gotpm keeps on disk between runs.
package cache

import (
	"fmt"

	"charm.land/log/v2"
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
}

// Clear removes the cloned remote repositories and the package index cache.
func Clear(opts *Options, log *log.Logger) error {
	remotesDir, err := remote.CacheDir()
	if err != nil {
		return err
	}
	cachePath, err := index.CachePath()
	if err != nil {
		return err
	}
	log.Debug("clearing", "remotes", remotesDir, "index", cachePath)

	size, err := paths.Size(remotesDir, cachePath)
	if err != nil {
		return err
	}

	if opts.DryRun {
		ui.Warnf("dry-run, would clear %s (remotes: %q, index cache: %q)", format(size), remotesDir, cachePath)
		return nil
	}

	if err := remote.ClearCache(); err != nil {
		return err
	}
	if err := index.ClearCache(); err != nil {
		return err
	}

	ui.Infof("cleared %s (remotes: %q, index cache: %q)", format(size), remotesDir, cachePath)
	return nil
}

func format(size int64) string {
	return fmt.Sprintf("%.1fMB", float64(size)/bytesPerMB)
}
