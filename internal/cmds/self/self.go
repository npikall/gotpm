// Package self implements the self commands: reporting what this binary is,
// and replacing it with a newer release.
package self

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	ui.Printf("version=%s\nhash=%s\nos=%s\narch=%s\ninstaller=%s\n",
		info.Version, info.Commit, info.OS, info.Arch, info.Installer)
}

func isHomebrewInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return isHomebrewCellarPath(exe)
}

func isHomebrewCellarPath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/Cellar/")
}

func isBrewInstall(info BuildInfo) bool {
	return info.Installer == "brew" || isHomebrewInstall()
}

func assetFilter() string {
	return assetFilterFor(runtime.GOOS, runtime.GOARCH)
}

func assetFilterFor(goos, goarch string) string {
	filter := fmt.Sprintf(`_%s_%s`, goos, goarch)
	if goos == "windows" {
		return filter + `\.exe$`
	}
	return filter + "$"
}

// Update replaces this binary with the latest release from GitHub. A Homebrew
// install is detected from the executable's path and sent to brew instead
// (ADR 0004).
func Update(info BuildInfo, opts *Options) error {
	ctx := context.Background()

	if info.IsDevelopment() {
		return ErrUpdateDevelopmentBuild
	}

	if isBrewInstall(info) {
		return ErrUpdateBrewBuild
	}

	currentVersion := strings.TrimPrefix(info.Version, "v")

	updater, err := selfupdate.NewUpdater(selfupdate.Config{Filters: []string{assetFilter()}})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	release, found, err := detectLatest(ctx, updater)
	if err != nil {
		return err
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

	err = ui.Spin("Downloading update...", func() error {
		if _, updateErr := updater.UpdateSelf(ctx, currentVersion, selfupdate.ParseSlug(repository)); updateErr != nil {
			return fmt.Errorf("update failed: %w", updateErr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	ui.Infof("updated gotpm %s → %s",
		ui.AccentBold.Render(info.Version),
		ui.AccentBold.Render("v"+latestVersion))
	return nil
}

// detectLatest looks up the newest release for this platform, with a spinner
// showing while it asks.
func detectLatest(ctx context.Context, updater *selfupdate.Updater) (*selfupdate.Release, bool, error) {
	var release *selfupdate.Release
	var found bool
	err := ui.Spin("Checking for updates...", func() error {
		var err error
		release, found, err = updater.DetectLatest(ctx, selfupdate.ParseSlug(repository))
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}
		return nil
	})
	return release, found, err
}
