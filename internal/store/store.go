// Package store manages the on-disk tree of installed typst packages, laid out
// as <root>/<namespace>/<name>/<version>.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/pkgfiles"
)

var (
	ErrAlreadyInstalled = errors.New("package already installed at destination")
	ErrNotInstalled     = errors.New("package not installed")
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

// Has reports whether something is already installed at the package's
// destination. A symlink counts as installed even when its target is gone.
func (s Store) Has(ref pkg.Ref) bool {
	return exists(s.Dir(ref))
}

// HasPackage reports whether any version of a package is installed.
func (s Store) HasPackage(namespace, name string) bool {
	return exists(s.PackageDir(namespace, name))
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
