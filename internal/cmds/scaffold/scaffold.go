// Package scaffold implements the init command: it writes the files a minimal
// typst package consists of.
//
// The package is not called init, since that name cannot be used to qualify an
// identifier in Go.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// LibFile is the entrypoint a new package starts out with.
var LibFile = []byte("#let greet(name) = [Hello #name]")

// Run creates a new package in the working directory, or in a new
// sub-directory of that name when one is given.
func Run(name string, log *log.Logger) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get the current working directory: %w", err)
	}

	if name != "" {
		dir = filepath.Join(dir, name)
		if err := os.Mkdir(dir, paths.DirPerm); err != nil {
			return fmt.Errorf("could not create directory: %w", err)
		}
	} else {
		name = filepath.Base(dir)
	}
	log.Debug("working directory", "current", dir)
	log.Debug("new package", "name", name)

	files := []struct {
		path    string
		content []byte
	}{
		{path: filepath.Join(dir, "typst.toml"), content: manifestFor(name)},
		{path: filepath.Join(dir, "lib.typ"), content: LibFile},
	}
	for _, file := range files {
		if err := paths.WriteFile(file.path, file.content); err != nil {
			return err
		}
	}

	ui.Infof("initialize package %q", name)
	return nil
}

func manifestFor(name string) []byte {
	return fmt.Appendf(nil, `[package]
name = "%s"
version = "0.1.0"
entrypoint = "lib.typ"`, name)
}
