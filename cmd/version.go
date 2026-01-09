/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/npikall/gotpm/internal/system"
	"github.com/npikall/gotpm/internal/version"
	"github.com/spf13/cobra"
)

var (
	short bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Manage the version of a Typst Package",
	Long:  `Use this command to change the version of the Package or to display it.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd := Must(os.Getwd())
		pkg := Must(system.OpenTypstTOML(cwd))
		pkgVersion := Must(version.ParseVersion(pkg.Version))

		if !short {
			fmt.Printf("%s %s\n", pkg.Name, InfoStyle.Render(pkgVersion.String()))
		} else {
			fmt.Printf("%s\n", InfoStyle.Render(pkgVersion.String()))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "show only the version number")
}
