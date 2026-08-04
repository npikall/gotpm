package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal"
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
	Package PackageMeta `toml:"package"`
}

type PackageMeta struct {
	Name       string `toml:"name"`
	Version    string `toml:"version"`
	Entrypoint string `toml:"entrypoint"`
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

func Load() (*Manifest, error) {
	manifest := &Manifest{}
	cwd, err := os.Getwd()
	if err != nil {
		return manifest, fmt.Errorf("could not get the current wooring directory: %w", err)
	}

	path, err := FindFile(cwd)
	if err != nil {
		return manifest, err
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
	if err := internal.UpdateTOML(&buf, internal.PackageMeta(manifest.Package), content, indent); err != nil {
		return fmt.Errorf("could not update typst.toml: %w", err)
	}

	if err := internal.WriteFile(file, buf.Bytes()); err != nil {
		return err //nolint: wrapcheck
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
