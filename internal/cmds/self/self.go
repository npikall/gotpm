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

// isHomebrewInstall reports whether the running binary lives under a
// Homebrew Cellar — the macOS Intel (/usr/local), macOS Apple Silicon
// (/opt/homebrew) and Linuxbrew (/home/linuxbrew/.linuxbrew) prefixes all
// share that one path segment. Symlinks are resolved first since a
// Homebrew-installed binary is normally reached through a symlink in
// <prefix>/bin, not the Cellar path itself.
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

// isHomebrewCellarPath is the pure check behind isHomebrewInstall, split out
// so it can be tested without touching the filesystem.
func isHomebrewCellarPath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/Cellar/")
}

// isBrewInstall reports whether this binary should be treated as a Homebrew
// install for self-update's purposes. info.Installer == "brew" covers
// binaries built by the pre-GoReleaser Homebrew formula, which stamped that
// ldflag at build time; isHomebrewInstall covers every install since, where
// GoReleaser ships the same binary to every channel and a Homebrew install
// is only distinguishable at runtime by where it landed on disk.
func isBrewInstall(info BuildInfo) bool {
	return info.Installer == "brew" || isHomebrewInstall()
}

// assetFilter is the regexp passed to go-selfupdate to pick this platform's
// release asset.
func assetFilter() string {
	return assetFilterFor(runtime.GOOS, runtime.GOARCH)
}

// assetFilterFor is the pure logic behind assetFilter, split out so it can
// be tested for every OS regardless of which one the test runs on.
// GoReleaser names assets "gotpm_<tag>_<os>_<arch>[.exe]" (see
// .goreleaser.yaml); this matches on the "_<os>_<arch>" suffix rather than
// the version-bearing prefix, since the version being looked up isn't known
// in advance.
func assetFilterFor(goos, goarch string) string {
	filter := fmt.Sprintf(`_%s_%s`, goos, goarch)
	if goos == "windows" {
		return filter + `\.exe$`
	}
	return filter + "$"
}

// Update replaces this binary with the latest release from GitHub.
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
