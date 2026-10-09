// Package manifest finds, reads, validates and updates a typst package
// manifest (typst.toml).
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/paths"
	tomledit "github.com/npikall/toml-edit"
)

// FileName is the name of a typst package manifest.
const FileName = "typst.toml"

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
	Tool     Tool        `toml:"tool,omitempty"`
}

// Tool holds the sections the typst manifest format reserves for third-party
// tooling. The typst compiler ignores them.
type Tool struct {
	Gotpm Gotpm `toml:"gotpm,omitempty"`
}

// Gotpm is the section gotpm itself owns in a manifest.
type Gotpm struct {
	// Dependencies lists packages the way they are imported in typst source,
	// e.g. "@gotpm/cetz:0.3.1".
	Dependencies []string `toml:"dependencies,omitempty"`
	// Fonts lists the font families the project needs, by the name Typst's
	// font setting uses, e.g. "Open Sans".
	Fonts []string `toml:"fonts,omitempty"`
	// PrePublishHook lists shell commands publish runs in the package root
	// before it copies the package, e.g. to generate a thumbnail.
	PrePublishHook []string `toml:"pre-publish-hook,omitempty"`
	// PostPublishHook lists shell commands publish runs in the package root
	// once it is done, e.g. to remove what the pre-publish hook generated.
	PostPublishHook []string `toml:"post-publish-hook,omitempty"`
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
		candidate := filepath.Join(currentDir, FileName)
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

// Update writes the name, version and entrypoint of manifest into file. Only
// the values that differ change; every other byte stays as the author wrote
// it.
func Update(file string, manifest *Manifest) error {
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not read typst.toml: %w", err)
	}

	doc, err := tomledit.Parse(string(content))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidManifest, err)
	}
	if err := setPackage(doc, manifest.Package); err != nil {
		return fmt.Errorf("could not update typst.toml: %w", err)
	}
	return paths.WriteFile(file, []byte(doc.String()))
}

func setPackage(doc *tomledit.Document, p PackageMeta) error {
	fields := []struct{ key, value string }{
		{"name", p.Name},
		{"version", p.Version},
		{"entrypoint", p.Entrypoint},
	}
	for _, field := range fields {
		path := []string{"package", field.key}
		current, err := doc.GetString(path...)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidManifest, err)
		}
		if current == field.value {
			continue
		}
		if err := doc.Set(path, field.value); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidManifest, err)
		}
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
