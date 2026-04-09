package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const manifestFileName = "typst.toml"

var (
	ErrManifestNotFound = errors.New("not found 'typst.toml': not a typst package directory")
	ErrInvalidManifest  = errors.New("invalid 'typst.toml'")
)

type Manifest struct {
	Package PackageMeta `toml:"package"`
}

type PackageMeta struct {
	Name       string `toml:"name"`
	Version    string `toml:"version"`
	Entrypoint string `toml:"entrypoint"`
}

// ValidateVersion reports whether the package version is a valid semver string.
func (p *PackageMeta) ValidateVersion() bool {
	return IsSemVer(p.Version)
}

// Bump updates the package version by the given increment or sets it to an
// exact semver string.
func (p *PackageMeta) Bump(increment string) error {
	if IsSemVer(increment) {
		p.Version = increment
		return nil
	}
	v, err := ParseVersion(p.Version)
	if err != nil {
		return err
	}
	if err := v.Bump(increment); err != nil {
		return err
	}
	p.Version = v.String()
	return nil
}

// LoadManifest reads and validates the typst.toml in the given directory.
func LoadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, manifestFileName)
	raw, err := readManifestFile(path)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := parseManifest(raw)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func readManifestFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrManifestNotFound
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	return data, nil
}

func parseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %s", ErrInvalidManifest, err)
	}
	return m, nil
}

func validateManifest(m Manifest) error {
	var errs []error
	if m.Package.Name == "" {
		errs = append(errs, errors.New("missing required field: package.name"))
	}
	if m.Package.Version == "" {
		errs = append(errs, errors.New("missing required field: package.version"))
	}
	if m.Package.Entrypoint == "" {
		errs = append(errs, errors.New("missing required field: package.entrypoint"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidManifest, errors.Join(errs...))
	}
	return nil
}
