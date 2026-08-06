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

var ErrAlreadyInstalled = errors.New("package already installed at destination")

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

// Has reports whether something is already installed at the package's
// destination. A symlink counts as installed even when its target is gone.
func (s Store) Has(ref pkg.Ref) bool {
	_, err := os.Lstat(s.Dir(ref))
	return err == nil
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
	if err := paths.Remove(s.Dir(ref)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
