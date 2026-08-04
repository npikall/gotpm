package bump

import (
	"errors"
	"fmt"
	"os"

	lg "charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/semver"
	"github.com/npikall/gotpm/internal/style"
	"github.com/npikall/gotpm/internal/write"
	"github.com/spf13/cobra"
)

type Options struct {
	DryRun   bool
	ShowCur  bool
	ShowNext bool
	Indent   bool
}

func GetOptions(cmd *cobra.Command) *Options {
	dryRun := internal.Must(cmd.Flags().GetBool("dry-run"))
	indent := internal.Must(cmd.Flags().GetBool("indent"))
	showCur := internal.Must(cmd.Flags().GetBool("show-current"))
	showNext := internal.Must(cmd.Flags().GetBool("show-next"))
	return &Options{
		DryRun:   dryRun,
		ShowCur:  showCur,
		ShowNext: showNext,
		Indent:   indent,
	}
}

var (
	ErrMissingArgument = errors.New("argument must be provided, can be one of [major|minor|patch] or a valid semver")
	ErrInvalidVersion  = errors.New("invalid version")
)

func Run(increment string, opts *Options, log *log.Logger) error {
	cwd := internal.Must(os.Getwd())
	manifestFile, err := manifest.FindFile(cwd)
	if err != nil {
		return err
	}
	log.Debug("load", "manifest", manifestFile)

	m, err := manifest.LoadFile(manifestFile)
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
	err = manifest.Update(manifestFile, m, opts.Indent)
	if err != nil {
		return fmt.Errorf("could not update %q: %w", manifestFile, err)
	}

	write.Infof(
		"updated version %s -> %s",
		style.AccentBold.Render(oldVersion),
		style.AccentBold.Render(m.Package.Version),
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
