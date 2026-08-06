// Package manifest finds, reads, validates and updates a typst package
// manifest (typst.toml).
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/paths"
)

const manifestFileName = "typst.toml"

var (
	ErrInvalidManifest   = errors.New("invalid 'typst.toml'")
	ErrMissingName       = errors.New("missing required field: package.name")
	ErrMissingVersion    = errors.New("missing required field: package.version")
	ErrMissingEntrypoint = errors.New("missing required field: package.entrypoint")
	ErrManifestNotFound  = errors.New("manifest file not found")
)

type Manifest struct {
	Package  PackageMeta `toml:"package"`
	Template Template    `toml:"template,omitempty"`
}

type PackageMeta struct {
	// required for typst compiler
	Name       string `toml:"name"`
	Version    string `toml:"version"`
	Entrypoint string `toml:"entrypoint"`

	// required for submissions to typst/packages
	Authors     []string `toml:"authors,omitempty"`
	License     string   `toml:"license,omitempty"`
	Description string   `toml:"description,omitempty"`

	// optional fields
	Homepage    string   `toml:"homepage,omitempty"`
	Repository  string   `toml:"repository,omitempty"`
	Keywords    []string `toml:"keywords,omitempty"`
	Categories  []string `toml:"categories,omitempty"`
	Disciplines []string `toml:"disciplines,omitempty"`
	Compiler    string   `toml:"compiler,omitempty"`
	Exclude     []string `toml:"exclude,omitempty"`
}

type Template struct {
	Path       string `toml:"path,omitempty"`
	Entrypoint string `toml:"entrypoint,omitempty"`
	Thumbnail  string `toml:"thumbnail,omitempty"`
}

func FindFile(dir string) (string, error) {
	currentDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("could not get current directory: %w", err)
	}

	for {
		candidate := filepath.Join(currentDir, manifestFileName)
		if err := paths.FileExists(candidate); err == nil {
			return candidate, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("%w: searching from %q", ErrManifestNotFound, dir)
		}
		currentDir = parentDir
	}
}

// Load reads the manifest of the package the current working directory
// belongs to.
func Load() (*Manifest, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return &Manifest{}, fmt.Errorf("could not get the current working directory: %w", err)
	}
	return LoadFrom(cwd)
}

// LoadFrom reads the manifest of the package dir belongs to, searching dir and
// then its parents.
func LoadFrom(dir string) (*Manifest, error) {
	path, err := FindFile(dir)
	if err != nil {
		return &Manifest{}, err
	}
	return LoadFile(path)
}

func LoadFile(path string) (*Manifest, error) {
	manifest := &Manifest{}

	data, err := os.ReadFile(path) //nolint: gosec
	if err != nil {
		return manifest, fmt.Errorf("could not read %q: %w", path, err)
	}

	if err := toml.Unmarshal(data, manifest); err != nil {
		return manifest, fmt.Errorf("%w: %w", ErrInvalidManifest, err)
	}

	if err := validateManifest(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func Update(file string, manifest *Manifest, indent bool) error {
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not read typst.toml: %w", err)
	}

	var buf bytes.Buffer
	if err := writeTOML(&buf, manifest.Package, content, indent); err != nil {
		return fmt.Errorf("could not update typst.toml: %w", err)
	}

	if err := paths.WriteFile(file, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// writeTOML writes the package metadata (name, version, entrypoint) back into
// the TOML document represented by data. Everything else in the document is
// carried over untouched.
func writeTOML(w io.Writer, p PackageMeta, data []byte, indent bool) error {
	var m map[string]any
	if err := toml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidManifest, err)
	}
	pkg, ok := m["package"].(map[string]any)
	if !ok {
		return ErrInvalidManifest
	}
	pkg["version"] = p.Version
	pkg["name"] = p.Name
	pkg["entrypoint"] = p.Entrypoint

	encoder := toml.NewEncoder(w)
	if !indent {
		encoder.Indent = ""
	}
	if err := encoder.Encode(m); err != nil {
		return fmt.Errorf("could not encode manifest: %w", err)
	}
	return nil
}

func validateManifest(m *Manifest) error {
	var errs []error
	if m.Package.Name == "" {
		errs = append(errs, ErrMissingName)
	}
	if m.Package.Version == "" {
		errs = append(errs, ErrMissingVersion)
	}
	if m.Package.Entrypoint == "" {
		errs = append(errs, ErrMissingEntrypoint)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidManifest, errors.Join(errs...))
	}
	return nil
}
