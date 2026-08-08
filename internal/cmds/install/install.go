// Package install implements the install command: it places a typst package
// into the local package store, either as a copy or as a symlink.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/remote"
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

// Run installs the package found at path, which may be empty to install the
// package of the current working directory.
func Run(path string, opts *Options, log *log.Logger) error {
	sourceDir, err := resolveSourceDir(path, opts, log)
	if err != nil {
		return err
	}
	log.Debug("operating in source", "path", sourceDir)

	manifestFile, err := manifest.FindFile(sourceDir)
	if err != nil {
		return fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return fmt.Errorf("could not load manifest: %w", err)
	}
	sourceDir = filepath.Dir(manifestFile)
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
		if err := s.Link(ref, sourceDir); err != nil {
			return err
		}
		ui.Infof("installed %s (editable)", ui.Package(ref.String()))
		return nil
	}

	spin := ui.Spinner("")
	spin.Start()
	err = s.Install(ref, sourceDir)
	spin.Stop()
	if err != nil {
		return err
	}
	ui.Infof("installed %s", ui.Package(ref.String()))
	return nil
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
		dir, cloned, err := remote.Ensure(opts.Remote, opts.Revision)
		if err != nil {
			return "", err
		}
		log.Debug("resolved remote", "url", opts.Remote, "path", dir, "cloned", cloned)
		return dir, nil
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
