// Package store manages the on-disk tree of installed typst packages, laid out
// as <root>/<namespace>/<name>/<version>.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/pkgfiles"
)

var (
	ErrAlreadyInstalled = errors.New("package already installed at destination")
	ErrNotInstalled     = errors.New("package not installed")
	// ErrFlatStore is returned by operations that need the namespace layer of
	// the layout, which an overridden store directory does not have.
	ErrFlatStore = errors.New("the package directory points at a single package; there are no namespaces to delete")
)

const stagingPrefix = ".gotpm-staging-"

// Store is a directory holding installed packages, normally laid out as
// <root>/<namespace>/<name>/<version>. An overridden directory is the destination
// itself; callers need not know which, as every path goes through Dir.
type Store struct {
	root string
	flat bool
}

// Open resolves the store to operate on. An override makes the store flat.
func Open(override string) (Store, error) {
	root, overridden, err := paths.InstallDir(override)
	if err != nil {
		return Store{}, fmt.Errorf("could not resolve package directory: %w", err)
	}
	return Store{root: root, flat: overridden}, nil
}

// OpenPackageDir opens the shared package directory the Typst compiler resolves
// imports from, laid out by namespace, name and version. It ignores the install
// dir override, which receives one package's files directly (ADR 0003).
func OpenPackageDir() (Store, error) {
	root, err := paths.TypstPackagesDir()
	if err != nil {
		return Store{}, fmt.Errorf("could not resolve package directory: %w", err)
	}
	return At(root), nil
}

// At returns a store rooted at an explicit directory, laid out by namespace,
// name and version.
func At(root string) Store {
	return Store{root: root}
}

// Root returns the directory this store lives in.
func (s Store) Root() string {
	return s.root
}

// Flat reports whether the store directory is the destination itself rather
// than the root of a namespace/name/version layout.
func (s Store) Flat() bool {
	return s.flat
}

// Dir returns the directory a package version occupies in this store.
func (s Store) Dir(ref pkg.Ref) string {
	if s.flat {
		return s.root
	}
	return ref.Dir(s.root)
}

// PackageDir returns the directory holding every version of a package.
func (s Store) PackageDir(namespace, name string) string {
	if s.flat {
		return s.root
	}
	return filepath.Join(s.root, namespace, name)
}

// NamespaceDir returns the directory holding every package of a namespace. A
// flat store has no namespace layer, so it has no such directory and the
// empty string is returned.
func (s Store) NamespaceDir(namespace string) string {
	if s.flat {
		return ""
	}
	return filepath.Join(s.root, namespace)
}

// Has reports whether something is already installed at the package's
// destination. A symlink counts as installed even when its target is gone.
func (s Store) Has(ref pkg.Ref) bool {
	return exists(s.Dir(ref))
}

// HasPackage reports whether any version of a package is installed.
func (s Store) HasPackage(namespace, name string) bool {
	return exists(s.PackageDir(namespace, name))
}

// HasNamespace reports whether a namespace is present. A flat store never has
// one.
func (s Store) HasNamespace(namespace string) bool {
	return !s.flat && exists(s.NamespaceDir(namespace))
}

// Install copies a package from srcDir into the store. The copy is staged in a
// hidden sibling directory and moved into place in one step, so an interrupted
// install cannot leave a half-written package the compiler would import.
func (s Store) Install(ref pkg.Ref, srcDir string) error {
	dest := s.Dir(ref)
	parent := filepath.Dir(dest)
	if err := paths.EnsureDir(parent); err != nil {
		return err
	}

	if s.flat {
		return pkgfiles.CopyTree(srcDir, dest)
	}
	return stageAndMove(parent, srcDir, dest)
}

// stageAndMove copies srcDir into a hidden directory in parent, then moves it
// to dest in one step.
func stageAndMove(parent, srcDir, dest string) error {
	staging, err := os.MkdirTemp(parent, stagingPrefix)
	if err != nil {
		return fmt.Errorf("could not create staging directory in %q: %w", parent, err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := fill(staging, srcDir); err != nil {
		return err
	}
	if err := os.Rename(staging, dest); err != nil {
		return fmt.Errorf("could not move staged package into %q: %w", dest, err)
	}
	return nil
}

// fill makes the staging directory readable like any other package directory
// and copies srcDir into it.
func fill(staging, srcDir string) error {
	if err := os.Chmod(staging, paths.DirPerm); err != nil {
		return fmt.Errorf("could not set permissions on %q: %w", staging, err)
	}
	return pkgfiles.CopyTree(srcDir, staging)
}

// Link installs a package as a symlink to srcDir, so edits to the source are
// picked up without reinstalling.
func (s Store) Link(ref pkg.Ref, srcDir string) error {
	dest := s.Dir(ref)
	if err := paths.EnsureDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("creating parent directory for symlink %q: %w", dest, err)
	}
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("resolving absolute path for symlink target %q: %w", srcDir, err)
	}
	if err := os.Symlink(absSrc, dest); err != nil {
		return fmt.Errorf("creating symlink %q -> %q: %w", dest, absSrc, err)
	}
	return nil
}

// Remove deletes an installed package version. A missing package is not an
// error.
func (s Store) Remove(ref pkg.Ref) error {
	return remove(s.Dir(ref))
}

// RemovePackage deletes every installed version of a package. A missing
// package is not an error.
func (s Store) RemovePackage(namespace, name string) error {
	return remove(s.PackageDir(namespace, name))
}

// RemoveNamespace deletes a namespace with every package in it. A missing
// namespace is not an error, but a flat store is: it has nothing that a
// namespace could name.
func (s Store) RemoveNamespace(namespace string) error {
	if s.flat {
		return ErrFlatStore
	}
	return remove(s.NamespaceDir(namespace))
}

// Namespace is one namespace of a store and the packages installed under it.
type Namespace struct {
	Name     string
	Packages []Package
}

// Package is one installed package and the versions of it that are present.
type Package struct {
	Name     string
	Versions []Version
}

// Version is one installed version of a package. Its name is the directory as
// found on disk, which is not necessarily valid semver.
type Version struct {
	Name     string
	Editable bool
}

// Exists reports whether the store directory is present.
func (s Store) Exists() bool {
	return isDir(s.root)
}

// Scan reports everything installed in the store, sorted by name. It is
// deliberately lenient: a version directory is listed under the name it has on
// disk, whether or not that name is valid semver.
func (s Store) Scan() ([]Namespace, error) {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read typst packages: %w", err)
	}
	return collect(s.root, entries, scanNamespace), nil
}

// collect scans each of the entries of dir, keeping what scan accepts. The
// entries come from os.ReadDir, so the result is sorted by name.
func collect[T any](dir string, entries []fs.DirEntry, scan func(dir string, entry fs.DirEntry) (T, bool)) []T {
	var out []T
	for _, entry := range entries {
		if item, ok := scan(dir, entry); ok {
			out = append(out, item)
		}
	}
	return out
}

// scanNamespace lists a namespace that holds at least one package.
func scanNamespace(root string, entry fs.DirEntry) (Namespace, bool) {
	if !entry.IsDir() || hidden(entry.Name()) {
		return Namespace{}, false
	}
	packages := scanPackages(filepath.Join(root, entry.Name()))
	return Namespace{Name: entry.Name(), Packages: packages}, len(packages) > 0
}

func scanPackages(namespaceDir string) []Package {
	entries, err := os.ReadDir(namespaceDir)
	if err != nil {
		return nil
	}
	return collect(namespaceDir, entries, scanPackage)
}

// scanPackage lists a package that holds at least one version.
func scanPackage(namespaceDir string, entry fs.DirEntry) (Package, bool) {
	if !entry.IsDir() || hidden(entry.Name()) {
		return Package{}, false
	}
	versions := scanVersions(filepath.Join(namespaceDir, entry.Name()))
	return Package{Name: entry.Name(), Versions: versions}, len(versions) > 0
}

func scanVersions(packageDir string) []Version {
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil
	}
	return collect(packageDir, entries, scanVersion)
}

// scanVersion lists a version directory, or a symlink to one for an editable
// install.
func scanVersion(packageDir string, entry fs.DirEntry) (Version, bool) {
	if hidden(entry.Name()) || !isDirFollowingLinks(filepath.Join(packageDir, entry.Name())) {
		return Version{}, false
	}
	return Version{Name: entry.Name(), Editable: entry.Type()&fs.ModeSymlink != 0}, true
}

func hidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func isDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func isDirFollowingLinks(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func remove(path string) error {
	if err := paths.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
