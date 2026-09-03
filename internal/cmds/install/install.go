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
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
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

var (
	ErrNoEditableRemote = errors.New("install remote package in `editable` mode is disallowed")
	// ErrUnresolvable is returned when a dependency declares a package whose
	// own lock has no entry for it. Unlike a missing lock entirely, this
	// means the dependant is broken, not merely unhelpful, so install stays
	// strict about it.
	ErrUnresolvable = errors.New("cannot resolve dependency")
	// ErrGraphInFlatStore is returned when a remote repository has
	// dependencies of its own but --install-dir names a destination that
	// holds one package's files, not a graph.
	ErrGraphInFlatStore = errors.New("--install-dir names a destination for one package's files")
)

// Run installs the package found at path, which may be empty to install the
// package of the current working directory.
func Run(path string, opts *Options, log *log.Logger) error {
	if opts.Editable && opts.Remote != "" {
		return ErrNoEditableRemote
	}
	if opts.Remote != "" {
		return runRemote(opts, log)
	}
	return runLocal(path, opts, log)
}

func runLocal(path string, opts *Options, log *log.Logger) error {
	sourceDir, err := resolveSourceDir(path)
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

func runRemote(opts *Options, log *log.Logger) error {
	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return err
	}

	walked, err := ui.WithSpinner("resolving "+opts.Remote, func() (depgraph.Result, error) {
		return depgraph.Walk(
			resolve.Request{URL: opts.Remote, Revision: opts.Revision},
			depgraph.Options{RootNamespace: opts.Namespace},
			log,
		)
	})
	if err != nil {
		return err
	}

	// Counting only resolved entries would let a repository whose one
	// dependency fails to resolve slip through as "single package" and get
	// vendored anyway — it has a dependency either way, resolved or not.
	dependencies := len(walked.Entries) - 1 + len(walked.Unresolved)
	if s.Flat() && dependencies > 0 {
		return fmt.Errorf("%w, but %s has %d dependencies"+
			"\nnote: omit --install-dir to install them into the package directory",
			ErrGraphInFlatStore, opts.Remote, dependencies)
	}
	warnings, err := splitUnresolved(walked.Unresolved)
	if err != nil {
		return err
	}

	installer := deps.Installer{Store: s, Logger: log, Force: opts.Force}
	results, err := ui.WithSpinner("installing", func() ([]deps.Result, error) {
		return installer.EnsureAll(walked.Entries)
	})
	if err != nil {
		return err
	}

	report(results)
	reportWarnings(warnings)
	return nil
}

// Unlike add, install writes no lock for anyone else to trust, so a
// dependant that ships no gotpm.lock at all is only a warning here; one whose
// lock is merely missing an entry is still a defect worth stopping for.
func splitUnresolved(unresolved []depgraph.Unresolved) ([]depgraph.Unresolved, error) {
	var warnings []depgraph.Unresolved
	for _, u := range unresolved {
		if u.Reason != depgraph.IncompleteLock {
			warnings = append(warnings, u)
			continue
		}
		return nil, fmt.Errorf("%w %s required by %s: its %s has no entry for it"+
			"\nnote: %s must commit a %s recording where its dependencies come from",
			ErrUnresolvable, u.Dependency, u.RequiredBy, lockfile.FileName, u.RequiredBy, lockfile.FileName)
	}
	return warnings, nil
}

func report(results []deps.Result) {
	for i, result := range results {
		switch {
		case i == 0 && result.Outcome == deps.UpToDate:
			ui.Infof("%s is already installed", ui.Package(result.Ref.String()))
		case i == 0:
			ui.Infof("installed %s from %s", ui.Package(result.Ref.String()), result.Entry.URL)
		default:
			ui.Infof("  %s (via %s)", ui.Package(result.Ref.String()), via(result.Entry))
		}
		if warning := result.DriftWarning(); warning != "" {
			ui.Warnf("%s", warning)
		}
	}
}

func via(entry lockfile.Entry) string {
	if len(entry.RequiredBy) == 0 {
		return entry.URL
	}
	return entry.RequiredBy[0]
}

func reportWarnings(warnings []depgraph.Unresolved) {
	for _, u := range warnings {
		ui.Warnf("%s declares %s but ships no %s; not installed", u.RequiredByURL, u.Dependency, lockfile.FileName)
	}
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

// resolveSourceDir determines which directory holds the package to install:
// the given path, or the working directory when path is empty.
func resolveSourceDir(path string) (string, error) {
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
