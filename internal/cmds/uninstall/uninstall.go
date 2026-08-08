// Package uninstall implements the uninstall command: it removes a package
// from the local package store, either one version, all of them, or a whole
// namespace.
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

var (
	ErrInsufficientPackage = errors.New("both package and version must be specified")
	// ErrNeedsConfirmation is returned when a namespace would be deleted with
	// nobody present to approve it.
	ErrNeedsConfirmation = errors.New("refusing to delete a namespace without a terminal to confirm it: pass --yes")
)

// Options holds the resolved uninstall flags.
type Options struct {
	Namespace string
	// NamespaceSet reports whether the namespace was named on the command
	// line rather than defaulted to. A namespace named on its own is the
	// request to delete all of it.
	NamespaceSet bool
	Version      string
	All          bool
	DryRun       bool
	// Yes approves the namespace deletion up front, without asking.
	Yes        bool
	InstallDir string
	// Confirm asks the user to approve a namespace deletion. Nil asks on the
	// terminal.
	Confirm func(question string) (bool, error)
}

// Run removes the named package, or the package of the current working
// directory when name is empty. A namespace given on its own removes the
// namespace as a whole.
func Run(name string, opts *Options, log *log.Logger) error {
	log.Debug("run flags", "namespace", opts.Namespace, "all", opts.All, "dry-run", opts.DryRun)

	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return err
	}

	// Decided before the manifest is read, so deleting a namespace works from
	// any directory rather than only from inside a package.
	if wipesNamespace(name, opts) {
		return removeNamespace(s, opts, log)
	}

	name, version, err := resolveIdentity(name, opts)
	if err != nil {
		return err
	}
	log.Debug("resolved package", "name", name, "version", version)

	// --all without a version removes the whole package; with one it stays
	// scoped to that version.
	if opts.All && version == "" {
		return removePackage(s, name, opts, log)
	}
	return removeVersion(s, name, version, opts, log)
}

// wipesNamespace reports whether the invocation names a namespace and nothing
// else. Anything further — a package, a version, --all — narrows the request
// back to a package inside that namespace.
func wipesNamespace(name string, opts *Options) bool {
	return name == "" && opts.NamespaceSet && opts.Version == "" && !opts.All
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

func removeNamespace(s store.Store, opts *Options, log *log.Logger) error {
	// An overridden package directory is a single destination, so the
	// namespace names nothing there. Checked before anything is reported or
	// asked, so a stray -n cannot delete the override.
	if s.Flat() {
		return store.ErrFlatStore
	}

	target := s.NamespaceDir(opts.Namespace)
	log.Debug("uninstalling namespace from", "path", target)

	if !s.HasNamespace(opts.Namespace) {
		return fmt.Errorf("%w: no namespace at %q", store.ErrNotInstalled, target)
	}

	packages, versions := count(s, opts.Namespace, log)
	if opts.DryRun {
		ui.Warnf("dryrun would delete %q: %d packages, %d versions", target, packages, versions)
		return nil
	}

	ui.Warnf("will delete %s: %d packages, %d versions",
		ui.Package(namespaceRef(opts.Namespace)), packages, versions)
	approved, err := approve(opts)
	if err != nil {
		return err
	}
	if !approved {
		ui.Infof("kept %s", ui.Package(namespaceRef(opts.Namespace)))
		return nil
	}

	if err := s.RemoveNamespace(opts.Namespace); err != nil {
		return err
	}
	ui.Infof("uninstalled %s", ui.Package(namespaceRef(opts.Namespace)))
	return nil
}

// approve reports whether the namespace may be deleted. --yes answers ahead
// of time; otherwise somebody has to be there to answer.
func approve(opts *Options) (bool, error) {
	if opts.Yes {
		return true, nil
	}
	ask := opts.Confirm
	if ask == nil {
		if !ui.IsTerminal() {
			return false, ErrNeedsConfirmation
		}
		ask = ui.Confirm
	}
	return ask("delete the whole namespace?")
}

// count reports how many packages and versions a namespace holds, for the
// warning shown before it is deleted. Counting is only for the report, so a
// scan that fails costs the numbers rather than the command.
func count(s store.Store, namespace string, log *log.Logger) (int, int) {
	namespaces, err := s.Scan()
	if err != nil {
		log.Debug("could not count namespace contents", "err", err)
		return 0, 0
	}
	for _, ns := range namespaces {
		if ns.Name != namespace {
			continue
		}
		versions := 0
		for _, p := range ns.Packages {
			versions += len(p.Versions)
		}
		return len(ns.Packages), versions
	}
	return 0, 0
}

func namespaceRef(namespace string) string {
	return "@" + namespace
}

// resolveIdentity returns the name and version to uninstall. A name given on
// the command line needs a version or --all alongside it; without a name the
// name comes from the manifest of the working directory, and so does the
// version unless the command line carries one of its own.
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
	switch {
	case opts.Version != "":
		return m.Package.Name, opts.Version, nil
	case opts.All:
		// An empty version leaves --all to remove every version.
		return m.Package.Name, "", nil
	default:
		return m.Package.Name, m.Package.Version, nil
	}
}
