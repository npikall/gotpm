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

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: initRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

var libFile = []byte("#let greet(name) = [Hello #name]")

func initRunner(cmd *cobra.Command, args []string) error {
	logger := setupLogger()
	cwd := Must(os.Getwd())

	var pkgName string
	if len(args) > 0 {
		pkgName = args[0]
	} else {
		pkgName = filepath.Base(cwd)
	}

	// Write minimal typst.toml
	bootstrap := []struct {
		path    string
		content []byte
	}{
		{path: "typst.toml", content: fmt.Appendf(nil, `[package]
name = "%s"
version = "0.1.0"
entrypoint = "lib.typ"`, pkgName)},
		{path: "lib.typ", content: libFile},
	}

	for _, boot := range bootstrap {
		err := os.WriteFile(boot.path, []byte(boot.content), 0644)
		if err != nil {
			return err
		}
	}

	logger.Info("initialize", "package", pkgName)
	return nil
}
