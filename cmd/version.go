/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/npikall/gotpm/internal/echo"
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
		cwd, err := os.Getwd()
		check(err)
		pkg, err := system.OpenTypstTOML(cwd)
		pkgVersion, err := version.ParseVersion(pkg.Version)
		if err != nil {
			echo.EchoErrorf("%s", err)
		}

		if !short {
			fmt.Fprintf(os.Stdout, "%s: %s\n", pkg.Name, pkgVersion)
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", pkgVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")
	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "show only the version number")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Panic when an unexpected error occurs
func check(e error) {
	if e != nil {
		panic(e)
	}
}
