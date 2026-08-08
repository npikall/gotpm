// Package bump implements the bump command: it changes the version recorded in
// a package manifest.
package bump

import (
	"errors"
	"fmt"
	"os"

	lg "charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/semver"
	"github.com/npikall/gotpm/internal/ui"
)

type Options struct {
	DryRun   bool
	ShowCur  bool
	ShowNext bool
	Indent   bool
}

var (
	ErrMissingArgument = errors.New("argument must be provided, can be one of [major|minor|patch] or a valid semver")
	ErrInvalidVersion  = errors.New("invalid version")
)

func Run(increment string, opts *Options, log *log.Logger) error {
	if increment == "" && !opts.ShowCur {
		return ErrMissingArgument
	}

	manifestFile, m, err := load(log)
	if err != nil {
		return err
	}

	oldVersion := m.Package.Version
	log.Debug("manifest", "version", oldVersion)
	if opts.ShowCur {
		_, _ = lg.Println(m.Package.Version)
		return nil
	}

	newVersion, err := bumpVersion(oldVersion, increment)
	if err != nil {
		return fmt.Errorf("could not bump version: %w", err)
	}
	log.Debug("setting", "version", newVersion)

	if opts.DryRun {
		log.Warn("performing dry-run")
		_, _ = lg.Printf("updated version %s -> %s", oldVersion, newVersion)
		return nil
	}

	if opts.ShowNext {
		_, _ = lg.Println(newVersion)
		return nil
	}

	m.Package.Version = newVersion
	if err := manifest.Update(manifestFile, m, opts.Indent); err != nil {
		return fmt.Errorf("could not update %q: %w", manifestFile, err)
	}

	ui.Infof(
		"updated version %s -> %s",
		ui.AccentBold.Render(oldVersion),
		ui.AccentBold.Render(newVersion),
	)
	return nil
}

// load reads the manifest of the package the working directory belongs to,
// returning its path alongside it so it can be written back.
func load(log *log.Logger) (string, *manifest.Manifest, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("could not get the current working directory: %w", err)
	}
	manifestFile, err := manifest.FindFile(cwd)
	if err != nil {
		return "", nil, err
	}
	log.Debug("load", "manifest", manifestFile)

	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return "", nil, err
	}
	return manifestFile, m, nil
}

func bumpVersion(version, increment string) (string, error) {
	v, err := semver.Parse(version)
	if err != nil {
		return "", fmt.Errorf("could not parse version: %w", err)
	}

	if semver.IsValidVersion(increment) {
		return increment, nil
	}

	err = v.Bump(increment)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}
