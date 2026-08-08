// Package self implements the self commands: reporting what this binary is,
// and replacing it with a newer release.
package self

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/npikall/gotpm/internal/ui"
)

const repository = "npikall/gotpm"

var (
	ErrUpdateDevelopmentBuild = errors.New("cannot self-update a development build; install a tagged release first")
	ErrUpdateBrewBuild        = errors.New("gotpm was installed via Homebrew; update with: brew upgrade gotpm")
)

// BuildInfo describes the running binary. It is filled from the values
// stamped into the cmd package at build time.
type BuildInfo struct {
	Version   string
	Commit    string
	OS        string
	Arch      string
	Installer string
}

// IsDevelopment reports whether this binary was built without a release tag.
func (b BuildInfo) IsDevelopment() bool {
	return b.Version == "dev"
}

// Options holds the resolved self-update flags.
type Options struct {
	// CheckOnly looks for an update without installing it.
	CheckOnly bool
}

// Version prints what this binary is.
func Version(info BuildInfo) {
	ui.Infof("gotpm version=%s hash=%s os=%s arch=%s installer=%s",
		info.Version, info.Commit, info.OS, info.Arch, info.Installer)
}

// Update replaces this binary with the latest release from GitHub.
func Update(info BuildInfo, opts *Options) error {
	ctx := context.Background()

	if info.IsDevelopment() {
		return ErrUpdateDevelopmentBuild
	}

	if info.Installer == "brew" {
		return ErrUpdateBrewBuild
	}

	currentVersion := strings.TrimPrefix(info.Version, "v")

	filter := fmt.Sprintf("gotpm-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		filter += ".exe"
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{Filters: []string{filter}})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	spin := ui.Spinner(" Checking for updates...")
	spin.Start()
	release, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repository))
	spin.Stop()

	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !found {
		ui.Warnf("no release found for %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}

	latestVersion := release.Version()
	if latestVersion == currentVersion {
		ui.Infof("already up to date (%s)", info.Version)
		return nil
	}

	if opts.CheckOnly {
		ui.Infof("update available: %s → %s",
			ui.AccentBold.Render(info.Version),
			ui.AccentBold.Render("v"+latestVersion))
		return nil
	}

	spin = ui.Spinner(" Downloading update...")
	spin.Start()
	_, err = updater.UpdateSelf(ctx, currentVersion, selfupdate.ParseSlug(repository))
	spin.Stop()

	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	ui.Infof("updated gotpm %s → %s",
		ui.AccentBold.Render(info.Version),
		ui.AccentBold.Render("v"+latestVersion))
	return nil
}
