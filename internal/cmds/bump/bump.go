// Package bump implements the bump command: it changes the version recorded in
// a package manifest.
package bump

import (
	"errors"
	"fmt"

	lg "charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
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
	if err := checkArgument(increment, opts); err != nil {
		return err
	}

	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	log.Debug("load", "manifest", project.File)
	log.Debug("manifest", "version", project.Manifest.Package.Version)

	if opts.ShowCur {
		_, _ = lg.Println(project.Manifest.Package.Version)
		return nil
	}
	return bumpProject(project, increment, opts, log)
}

// checkArgument requires an increment unless only the current version is shown.
func checkArgument(increment string, opts *Options) error {
	if increment == "" && !opts.ShowCur {
		return ErrMissingArgument
	}
	return nil
}

func bumpProject(project *deps.Project, increment string, opts *Options, log *log.Logger) error {
	oldVersion := project.Manifest.Package.Version
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
	return setVersion(project, newVersion, opts.Indent)
}

func setVersion(project *deps.Project, newVersion string, indent bool) error {
	m := project.Manifest
	oldVersion := m.Package.Version
	m.Package.Version = newVersion
	if err := manifest.Update(project.File, m, indent); err != nil {
		return fmt.Errorf("could not update %q: %w", project.File, err)
	}

	ui.Infof(
		"updated version %s -> %s",
		ui.AccentBold.Render(oldVersion),
		ui.AccentBold.Render(newVersion),
	)
	return nil
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
