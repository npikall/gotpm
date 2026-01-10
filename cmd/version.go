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

	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/system"
	"github.com/npikall/gotpm/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Manage the version of a Typst Package",
	Long:  `Use this command to change the version of the Package or to display it.`,
	Run:   versionRunner,
}

const bumpDefault string = "none"

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("short", "s", false, "show only the version number")
	versionCmd.Flags().StringP("bump", "b", bumpDefault, "Bump the version. Can be on of [major, minor, patch]")
}

func versionRunner(cmd *cobra.Command, args []string) {
	cwd := Must(os.Getwd())
	pkg := Must(system.OpenTypstTOML(cwd))
	oldPkgVersion := pkg.Version
	pkgVersion := Must(version.ParseVersion(oldPkgVersion))

	short := Must(cmd.Flags().GetBool("short"))
	bump := Must(cmd.Flags().GetString("bump"))
	isBumping := bump != bumpDefault

	if !short && !isBumping {
		fmt.Printf("%s %s\n", pkg.Name, InfoStyle.Render(pkgVersion.String()))
		os.Exit(0)
	}
	if short && !isBumping {
		fmt.Printf("%s\n", InfoStyle.Render(pkgVersion.String()))
		os.Exit(0)
	}

	if isBumping {
		err := pkgVersion.Bump(bump)
		if err != nil {
			LogFatalf("%s", err)
		}
	}

	pkg.Version = pkgVersion.String()

	typstTOML := filepath.Join(cwd, "typst.toml")
	typstTOMLContent := Must(os.ReadFile(typstTOML))

	typstTOMLFile := Must(os.Open(typstTOML))
	defer typstTOMLFile.Close()

	err := manifest.WriteTOML(typstTOMLFile, pkg, typstTOMLContent)
	if err != nil {
		// LogFatalf("%s", err)
		panic(err)
	}
	LogInfof("updated version %s -> %s", oldPkgVersion, pkg.Version)
}
