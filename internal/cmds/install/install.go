// Package install implements the install command: it places a typst package
// into the package directory, either as a copy or as a symlink of the working
// tree, or by fetching a repository and everything it depends on.
package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

// Options holds the resolved install flags.
type Options struct {
	Force      bool
	Editable   bool
	Namespace  string
	Remote     string
	InstallDir string
	Revision   string
}

var ErrNoEditableRemote = errors.New("install remote package in `editable` mode is disallowed")

// Run installs the package found at path, which may be empty to install the
// package of the current working directory.
func Run(path string, opts *Options, log *log.Logger) error {
	if opts.Editable && opts.Remote != "" {
		return ErrNoEditableRemote
	}

	sourceDir, err := resolveSourceDir(path, opts, log)
	if err != nil {
		return err
	}
	log.Debug("operating in source", "path", sourceDir)

	sourceDir, m, err := findRootManifest(sourceDir)
	if err != nil {
		return err
	}
	log.Debug("found package", "name", m.Package.Name, "version", m.Package.Version, "root", sourceDir)

	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return err
	}
	ref, err := pkg.New(opts.Namespace, m.Package.Name, m.Package.Version)
	if err != nil {
		return err
	}
	log.Debug("resolved destination", "path", s.Dir(ref))

	if err := clearDestination(s, ref, opts.Force); err != nil {
		return err
	}

	if opts.Editable {
		return linkAndReport(s, ref, sourceDir)
	}

	if err := ui.Spin("", func() error { return s.Install(ref, sourceDir) }); err != nil {
		return err
	}
	ui.Infof("installed %s", ui.Package(ref.String()))
	return nil
}

func linkAndReport(s store.Store, ref pkg.Ref, src string) error {
	if err := s.Link(ref, src); err != nil {
		return err
	}
	ui.Infof("installed %s (editable)", ui.Package(ref.String()))
	return nil
}

func findRootManifest(src string) (string, *manifest.Manifest, error) {
	manifestFile, err := manifest.FindFile(src)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return "", nil, fmt.Errorf("could not load manifest: %w", err)
	}
	sourceDir := filepath.Dir(manifestFile)
	return sourceDir, m, nil
}

// clearDestination removes an existing install when force is set, and rejects
// the install otherwise.
func clearDestination(s store.Store, ref pkg.Ref, force bool) error {
	if !s.Has(ref) {
		return nil
	}
	if !force {
		return fmt.Errorf("%w: %q", store.ErrAlreadyInstalled, s.Dir(ref))
	}
	if err := s.Remove(ref); err != nil {
		return fmt.Errorf("removing existing package: %w", err)
	}
	return nil
}

// resolveSourceDir determines which directory holds the package to install: a
// clone of a remote repository, the given path, or the working directory.
func resolveSourceDir(path string, opts *Options, log *log.Logger) (string, error) {
	if opts.Remote != "" {
		resolved, err := resolve.Resolve(resolve.Request{URL: opts.Remote, Revision: opts.Revision}, log)
		if err != nil {
			return "", err
		}
		return resolved.Dir, nil
	}
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current working directory: %w", err)
		}
		return cwd, nil
	}
	return resolveProvidedPath(path)
}

func resolveProvidedPath(rawPath string) (string, error) {
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", rawPath, err)
	}
	if err := paths.DirectoryExists(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}
