package deps

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
)

// Project is the typst project a command operates on: the directory holding
// its typst.toml, and with it the gotpm.lock that sits beside it.
//
// The two files split the job between them. typst.toml says which packages the
// project wants, in the form its source imports them by; gotpm.lock says where
// each one came from and at which commit. Neither is enough on its own, which
// is why every dependency command reads both.
type Project struct {
	Dir      string
	File     string
	Manifest *manifest.Manifest
}

// OpenProject finds the project the working directory belongs to.
func OpenProject() (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get the current working directory: %w", err)
	}
	return OpenProjectAt(cwd)
}

// OpenProjectAt finds the project dir belongs to, searching dir and then its
// parents.
func OpenProjectAt(dir string) (*Project, error) {
	file, err := manifest.FindFile(dir)
	if err != nil {
		if errors.Is(err, manifest.ErrManifestNotFound) {
			return nil, fmt.Errorf("%w\nnote: run 'gotpm init' to start a project here", err)
		}
		return nil, err
	}
	m, err := manifest.LoadFile(file)
	if err != nil {
		return nil, err
	}
	return &Project{Dir: filepath.Dir(file), File: file, Manifest: m}, nil
}

// Dependencies returns the packages the project declares, in the order they
// are written in the manifest.
func (p *Project) Dependencies() []string {
	return p.Manifest.Dependencies()
}

// SetDependencies rewrites the declared dependency list, keeping the rest of
// the manifest byte for byte as it was.
func (p *Project) SetDependencies(deps []string) error {
	if err := manifest.SetDependencies(p.File, deps); err != nil {
		return err
	}
	p.Manifest.Tool.Gotpm.Dependencies = deps
	return nil
}

// Lock reads the lock file committed next to the manifest. A project without
// one yet reads as an empty lock.
func (p *Project) Lock() (*lockfile.Lock, error) {
	return lockfile.Load(p.Dir)
}

// SaveLock writes the lock back.
func (p *Project) SaveLock(l *lockfile.Lock) error {
	return lockfile.Save(p.Dir, l)
}
