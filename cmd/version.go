/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"bytes"
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
	Use: "version",
	Example: `gotpm version
gotpm version --short
gotpm version --bump major
gotpm version --bump 0.1.2
`,
	Short: "Manage the version of a Typst Package",
	Long:  `Use this command to change the version of the Package or to display it.`,
	Run:   versionRunner,
}

const bumpDefault string = "none"

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("short", "s", false, "show only the version number")
	versionCmd.Flags().StringP("bump", "b", bumpDefault, "Bump the version. Can be on of [major, minor, patch] or a valid SemVer String (e.g.: 0.1.2)")
}

func versionRunner(cmd *cobra.Command, args []string) {
	cwd := Must(os.Getwd())
	pkg := Must(system.OpenTypstTOML(cwd))
	oldPkgVersion := pkg.Version
	newPkgVersion := Must(version.ParseVersion(oldPkgVersion))

	// Handle Runtime Flags
	short := Must(cmd.Flags().GetBool("short"))
	bump := Must(cmd.Flags().GetString("bump"))

	isBumping := bump != bumpDefault
	isFixedVersion := version.IsSemVer(bump)

	switch {
	case !short && !isBumping:
		fmt.Printf("%s %s\n", pkg.Name, InfoStyle.Render(newPkgVersion.String()))
		os.Exit(0)
	case short && !isBumping:
		fmt.Printf("%s\n", InfoStyle.Render(newPkgVersion.String()))
		os.Exit(0)
	case isFixedVersion:
		pkg.Version = bump
	case isBumping && !isFixedVersion:
		err := newPkgVersion.Bump(bump)
		if err != nil {
			LogFatalf("%s", err)
		}
		pkg.Version = newPkgVersion.String()
	}

	// Read the existing TOML file
	typstTOML := filepath.Join(cwd, "typst.toml")
	typstTOMLContent := Must(os.ReadFile(typstTOML))

	// Write updated TOML to a buffer first
	var buf bytes.Buffer
	err := manifest.WriteTOML(&buf, pkg, typstTOMLContent)
	if err != nil {
		LogFatalf("failed to update TOML: %s", err)
	}

	// Write the buffer to the file
	err = os.WriteFile(typstTOML, buf.Bytes(), 0644)
	if err != nil {
		LogFatalf("failed to write file: %s", err)
	}

	LogInfof("updated version %s -> %s", oldPkgVersion, pkg.Version)
}
