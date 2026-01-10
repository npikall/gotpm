/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/npikall/gotpm/internal/system"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall a Typst Package from the local Storage",
	Run:   uninstallRunner,
}

const versionFlagDefault string = "current"

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringP("namespace", "n", "local", "The namespace from which the package should be removed from.")
	uninstallCmd.Flags().StringP("version", "v", versionFlagDefault, "The specific version of a package that should be removed.")
}

func uninstallRunner(cmd *cobra.Command, args []string) {
	goos := runtime.GOOS
	homeDir := Must(os.UserHomeDir())
	cwd := Must(os.Getwd())

	pkg := Must(system.OpenTypstTOML(cwd))
	namespace := Must(cmd.Flags().GetString("namespace"))
	version := Must(cmd.Flags().GetString("version"))
	if version == "current" {
		version = pkg.Version
	}
	dst := Must(system.GetStoragePath(goos, homeDir, namespace, pkg.Name, version))

	err := os.RemoveAll(dst)
	if err != nil {
		log.Fatalln(err)
	}
	identifier := HighStyle.Render(fmt.Sprintf("%s:%s", pkg.Name, pkg.Version))
	LogInfof("Uninstalled %s from %s", identifier, namespace)
}
