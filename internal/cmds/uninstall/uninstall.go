// Package uninstall implements the uninstall command: it removes a package
// from the local package store, either one version or all of them.
package uninstall

import (
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

var ErrInsufficientPackage = errors.New("both package and version must be specified")

// Options holds the resolved uninstall flags.
type Options struct {
	Namespace  string
	Version    string
	All        bool
	DryRun     bool
	InstallDir string
}

// Run removes the named package, or the package of the current working
// directory when name is empty.
func Run(name string, opts *Options, log *log.Logger) error {
	log.Debug("run flags", "namespace", opts.Namespace, "all", opts.All, "dry-run", opts.DryRun)

	name, version, err := resolveIdentity(name, opts)
	if err != nil {
		return err
	}
	log.Debug("resolved package", "name", name, "version", version)

	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return err
	}

	// --all without a version removes the whole package; with one it stays
	// scoped to that version.
	if opts.All && version == "" {
		return removePackage(s, name, opts, log)
	}
	return removeVersion(s, name, version, opts, log)
}

func removeVersion(s store.Store, name, version string, opts *Options, log *log.Logger) error {
	ref, err := pkg.New(opts.Namespace, name, version)
	if err != nil {
		return err
	}
	target := s.Dir(ref)
	log.Debug("uninstalling from", "path", target)

	if !s.Has(ref) {
		return fmt.Errorf("%w: %q", store.ErrNotInstalled, target)
	}
	if opts.DryRun {
		ui.Warnf("dryrun would delete %q", target)
		return nil
	}
	if err := s.Remove(ref); err != nil {
		return err
	}
	ui.Infof("uninstalled %s", ui.Package(ref.String()))
	return nil
}

func removePackage(s store.Store, name string, opts *Options, log *log.Logger) error {
	target := s.PackageDir(opts.Namespace, name)
	log.Debug("uninstalling from", "path", target)

	if !s.HasPackage(opts.Namespace, name) {
		return fmt.Errorf("%w: %q", store.ErrNotInstalled, target)
	}
	if opts.DryRun {
		ui.Warnf("dryrun would delete %q", target)
		return nil
	}
	if err := s.RemovePackage(opts.Namespace, name); err != nil {
		return err
	}
	ui.Infof("uninstalled %s", ui.Package(fmt.Sprintf("@%s/%s:*.*.*", opts.Namespace, name)))
	return nil
}

// resolveIdentity returns the name and version to uninstall. A name given on
// the command line needs a version or --all alongside it; without a name both
// come from the manifest of the working directory.
func resolveIdentity(name string, opts *Options) (string, string, error) {
	if name != "" {
		if opts.Version == "" && !opts.All {
			return "", "", ErrInsufficientPackage
		}
		return name, opts.Version, nil
	}
	m, err := manifest.Load()
	if err != nil {
		return "", "", fmt.Errorf("could not load typst manifest: %w", err)
	}
	return m.Package.Name, m.Package.Version, nil
}
