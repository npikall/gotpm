// Package scaffold implements the init command: it writes the files a minimal
// Typst package, or a minimal document project, consists of.
//
// The package is not called init, since that name cannot be used to qualify an
// identifier in Go.
package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// Kind is what init scaffolds. A package is source other projects import and
// the typst compiler never renders on its own; a document project is source
// that is compiled into a document and that nothing ever imports. Both carry a
// manifest, because both may declare dependencies.
type Kind string

const (
	// KindPackage is the zero value, so an Options with no kind set scaffolds
	// a package. It is what a manifest without a kind is read as, and is
	// therefore never written to one.
	KindPackage Kind = "package"
	// KindDocument is recorded in the manifest, since nothing else in the
	// scaffolded files distinguishes a document project from a package.
	KindDocument Kind = "document"
)

// ErrUnknownKind guards the file selection below: a kind it does not know
// would otherwise write no files at all and report success.
var ErrUnknownKind = errors.New("unknown kind")

// LibFile is the entrypoint a new package starts out with, MainFile the one a
// new document project starts out with.
var (
	LibFile  = []byte("#let greet(name) = [Hello #name]")
	MainFile = []byte("= Hello\nWorld")
)

type Options struct {
	Kind Kind
}

// Run creates a new package in the working directory, or in a new
// subdirectory of that name when one is given.
func Run(name string, opts Options, log *log.Logger) error {
	kind, err := resolveKind(opts.Kind)
	if err != nil {
		return err
	}
	dir, name, err := projectDir(name)
	if err != nil {
		return err
	}
	log.Debug("working directory", "current", dir)
	log.Debug("new package", "name", name, "kind", kind)

	if err := writeFiles(filesFor(kind, dir, name)); err != nil {
		return err
	}
	ui.Infof("initialize %s %q", kind, name)
	return nil
}

// projectDir returns the directory to scaffold into and the project's name.
// A given name becomes a new subdirectory of the working directory; without
// one, the working directory itself is used and lends the project its name.
func projectDir(name string) (string, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("could not get the current working directory: %w", err)
	}
	if name == "" {
		return dir, filepath.Base(dir), nil
	}
	dir = filepath.Join(dir, name)
	if err := os.Mkdir(dir, paths.DirPerm); err != nil {
		return "", "", fmt.Errorf("could not create directory: %w", err)
	}
	return dir, name, nil
}

func writeFiles(files []file) error {
	for _, file := range files {
		if err := paths.WriteFile(file.path, file.content); err != nil {
			return err
		}
	}
	return nil
}

// resolveKind fills in the default an empty kind stands for, and rejects
// anything else it does not know.
func resolveKind(kind Kind) (Kind, error) {
	switch kind {
	case "":
		return KindPackage, nil
	case KindPackage, KindDocument:
		return kind, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
}

type file struct {
	path    string
	content []byte
}

func filesFor(kind Kind, dir, name string) []file {
	entrypoint, source := "lib.typ", LibFile
	if kind == KindDocument {
		entrypoint, source = "main.typ", MainFile
	}
	return []file{
		{
			path:    filepath.Join(dir, entrypoint),
			content: source,
		},
		{
			path:    filepath.Join(dir, manifest.FileName),
			content: manifestFor(name, entrypoint, kind),
		},
	}
}

// manifestFor writes the manifest of a new project. A document project needs
// its kind recorded; a package is what a manifest without one is read as, so
// it gets no [tool.gotpm] section at all.
func manifestFor(name, entrypoint string, kind Kind) []byte {
	m := fmt.Appendf(nil, `[package]
name = "%s"
version = "0.1.0"
entrypoint = "%s"`, name, entrypoint)

	if kind == KindDocument {
		m = fmt.Appendf(m, `

[tool.gotpm]
kind = "%s"`, kind)
	}
	return m
}
