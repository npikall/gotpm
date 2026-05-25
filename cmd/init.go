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

	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use: "init",
	Example: `# initialize a new Package
gotpm init`,
	Short: "Initialize a new minimal Typst Package",
	RunE:  initRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

var libFile = []byte("#let greet(name) = [Hello #name]")

func initRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)
	cwd := internal.Must(os.Getwd())

	var pkgName string
	if len(args) > 0 {
		pkgName = args[0]
		cwd = filepath.Join(cwd, pkgName)
		err := os.Mkdir(cwd, 0o755)
		if err != nil {
			return err
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
		{path: filepath.Join(cwd, "lib.typ"), content: libFile},
	}

	for _, boot := range bootstrap {
		err := os.WriteFile(boot.path, []byte(boot.content), 0o644)
		if err != nil {
			return err
		}
	}

	internal.PrintInfo("initialize package %q", pkgName)
	return nil
}
