/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use: "init",
	Example: `# initialize a new Package
gotpm init`,
	Short: "Initialize a new minimal Typst Package",
	RunE:  InitRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

var LibFile = []byte("#let greet(name) = [Hello #name]")

func InitRunner(cmd *cobra.Command, args []string) error {
	logger := newLogger(cmd)
	cwd := Must(os.Getwd())

	var pkgName string
	if len(args) > 0 {
		pkgName = args[0]
		cwd = filepath.Join(cwd, pkgName)
		if err := os.Mkdir(cwd, paths.DirPerm); err != nil {
			return fmt.Errorf("could not create directory: %w", err)
		}
	} else {
		pkgName = filepath.Base(cwd)
	}
	logger.Debug("working directory", "current", cwd)
	logger.Debug("new package", "name", pkgName)

	// Write minimal typst.toml
	bootstrap := []struct {
		path    string
		content []byte
	}{
		{path: filepath.Join(cwd, "typst.toml"), content: fmt.Appendf(nil, `[package]
name = "%s"
version = "0.1.0"
entrypoint = "lib.typ"`, pkgName)},
		{path: filepath.Join(cwd, "lib.typ"), content: LibFile},
	}

	for _, boot := range bootstrap {
		if err := paths.WriteFile(boot.path, boot.content); err != nil {
			return err
		}
	}

	ui.Infof("initialize package %q", pkgName)
	return nil
}
