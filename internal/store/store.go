// Package store manages the on-disk tree of installed typst packages, laid out
// as <root>/<namespace>/<name>/<version>.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

// Store is a directory holding installed packages.
//
// A store is normally laid out as <root>/<namespace>/<name>/<version>, but when
// the user overrides the directory (--install-dir or $GOTPM_INSTALL_DIR) the
// root itself is the destination. Callers do not need to know which of the two
// they got: every path goes through Dir.
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

// Install copies a package from srcDir into the store.
func (s Store) Install(ref pkg.Ref, srcDir string) error {
	dest := s.Dir(ref)
	if err := paths.EnsureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	return pkgfiles.CopyTree(srcDir, dest)
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
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read typst packages: %w", err)
	}

	var namespaces []Namespace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packages := scanPackages(filepath.Join(s.root, entry.Name()))
		if len(packages) == 0 {
			continue
		}
		namespaces = append(namespaces, Namespace{Name: entry.Name(), Packages: packages})
	}
	slices.SortFunc(namespaces, func(a, b Namespace) int { return strings.Compare(a.Name, b.Name) })
	return namespaces, nil
}

// scanPackages lists the packages of one namespace directory. Packages without
// any version directory are left out.
func scanPackages(namespaceDir string) []Package {
	entries, err := os.ReadDir(namespaceDir)
	if err != nil {
		return nil
	}

	var packages []Package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versions := scanVersions(filepath.Join(namespaceDir, entry.Name()))
		if len(versions) == 0 {
			continue
		}
		packages = append(packages, Package{Name: entry.Name(), Versions: versions})
	}
	slices.SortFunc(packages, func(a, b Package) int { return strings.Compare(a.Name, b.Name) })
	return packages
}

// scanVersions lists the versions of one package directory. A version that is
// a symlink was installed as editable.
func scanVersions(packageDir string) []Version {
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil
	}

	var versions []Version
	for _, entry := range entries {
		// Follows symlinks, so an editable install still counts as a version.
		if !isDirFollowingLinks(filepath.Join(packageDir, entry.Name())) {
			continue
		}
		versions = append(versions, Version{
			Name:     entry.Name(),
			Editable: entry.Type()&fs.ModeSymlink != 0,
		})
	}
	slices.SortFunc(versions, func(a, b Version) int { return strings.Compare(a.Name, b.Name) })
	return versions
}

func isDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func isDirFollowingLinks(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// exists reports whether path is present, counting a symlink whose target is
// gone as present.
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
