// Package pkg models a reference to a typst package: the namespace, name and
// version that together identify it in an import statement and on disk.
package pkg

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/npikall/gotpm/internal/semver"
)

var ErrInvalidRef = errors.New("not a valid package reference")

// Ref identifies one version of one package, e.g. "@preview/cetz:0.5.2".
type Ref struct {
	Namespace string
	Name      string
	Version   semver.Version
}

// New builds a Ref from its parts, rejecting a version that is not semver.
func New(namespace, name, version string) (Ref, error) {
	if namespace == "" {
		return Ref{}, fmt.Errorf("%w: missing namespace", ErrInvalidRef)
	}
	if name == "" {
		return Ref{}, fmt.Errorf("%w: missing package name", ErrInvalidRef)
	}
	v, err := semver.Parse(version)
	if err != nil {
		return Ref{}, fmt.Errorf("%w: %w", ErrInvalidRef, err)
	}
	return Ref{Namespace: namespace, Name: name, Version: *v}, nil
}

// ParseImport reads a reference in its import-statement form,
// e.g. "@preview/cetz:0.5.2". The leading "@" is optional.
func ParseImport(s string) (Ref, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(s), "@")
	namespace, identifier, found := strings.Cut(trimmed, "/")
	if !found {
		return Ref{}, fmt.Errorf("%w: %q is missing the %q separator", ErrInvalidRef, s, "/")
	}
	name, version, found := strings.Cut(identifier, ":")
	if !found {
		return Ref{}, fmt.Errorf("%w: %q is missing the %q separator", ErrInvalidRef, s, ":")
	}
	return New(namespace, name, version)
}

// String renders the reference the way it is written in an import statement.
func (r Ref) String() string {
	return "@" + r.Namespace + "/" + r.Name + ":" + r.Version.String()
}

// Segments returns the path segments a package occupies under a package
// directory: namespace, name and version.
func (r Ref) Segments() []string {
	return []string{r.Namespace, r.Name, r.Version.String()}
}

// Dir returns the directory this package version lives in below root.
func (r Ref) Dir(root string) string {
	return filepath.Join(append([]string{root}, r.Segments()...)...)
}
