/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	version "github.com/npikall/gotpm/internal/bump"
	"github.com/npikall/gotpm/internal/files"
	"github.com/spf13/cobra"
)

// bumpCmd represents the version command
var bumpCmd = &cobra.Command{
	Use: "bump [increment|version]",
	Example: `# bump with a given increment
gotpm bump major

# set to a specific version
gotpm bump 0.1.2
`,
	Short: "Manage the version of a Typst Package",
	Long: `Use this command to change the version of the Package or to display it.
Valid arguments can be:
	- major
	- minor
	- patch
	- a valid semantic version (e.g. 0.1.2)`,
	RunE: bumpRunner,
}

func init() {
	rootCmd.AddCommand(bumpCmd)
	bumpCmd.Flags().Bool("dry-run", false, "Perform a dry-run")
	bumpCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
	bumpCmd.Flags().BoolP("show-current", "c", false, "Show the version of the current package")
	bumpCmd.Flags().BoolP("show-next", "n", false, "Show the version of the package if it where bumped")
	bumpCmd.Flags().BoolP("indent", "i", false, "Use Indentation in the typst.toml file.")
}

var ErrMissingArgument = errors.New("argument must be provided, can be one of [major|minor|patch] or a valid semver")

func bumpRunner(cmd *cobra.Command, args []string) error {
	logger := setupVerboseLogger(cmd)

	cwd := Must(os.Getwd())
	logger.Debug("running in", "cwd", cwd)

	pkg, err := files.LoadPackageFromDirectory(cwd)
	if err != nil {
		return err
	}

	showCurrent := Must(cmd.Flags().GetBool("show-current"))
	if showCurrent {
		fmt.Println(pkg.Version)
		return nil
	}

	oldPkgVersion := pkg.Version
	logger.Debug("from 'typst.toml'", "version", oldPkgVersion)

	newPkgVersion, err := version.ParseVersion(oldPkgVersion)
	if err != nil {
		return err
	}

	var bumpArg string
	if len(args) > 0 {
		bumpArg = args[0]
	} else {
		return ErrMissingArgument
	}

	dryRun := Must(cmd.Flags().GetBool("dry-run"))
	isFixedVersion := version.IsSemVer(bumpArg)

	err = setVersionOrIncrement(isFixedVersion, pkg, bumpArg, newPkgVersion)
	if err != nil {
		return err
	}
	logger.Debug("setting toml", "version", pkg.Version)

	if dryRun {
		logger.Warn("performing dry-run")
		logger.Infof("updated version %s -> %s", oldPkgVersion, pkg.Version)
		return nil
	}

	showNext := Must(cmd.Flags().GetBool("show-next"))
	if showNext {
		fmt.Println(newPkgVersion)
		return nil
	}

	typstTOML := filepath.Join(cwd, "typst.toml")
	typstTOMLContent, err := os.ReadFile(typstTOML)
	if err != nil {
		return err
	}
	logger.Debug("editing", "file", typstTOML)

	// Write updated TOML to a buffer first
	indent := Must(cmd.Flags().GetBool("indent"))
	var buf bytes.Buffer
	err = files.UpdateTOML(&buf, pkg, typstTOMLContent, indent)
	if err != nil {
		return err
	}
	logger.Debug("write edited toml to buffer")

	// Write the buffer to the file
	err = os.WriteFile(typstTOML, buf.Bytes(), 0644)
	if err != nil {
		return err
	}
	logger.Debug("write buffer", "file", typstTOML)

	logger.Infof("updated version %s -> %s", oldPkgVersion, pkg.Version)
	return nil
}

func setVersionOrIncrement(isFixedVersion bool, pkg files.PackageInfo, bumpArg string, newPkgVersion version.Version) error {
	switch {
	case isFixedVersion:
		pkg.Version = bumpArg
	default:
		err := newPkgVersion.Bump(bumpArg)
		if err != nil {
			return err
		}
		pkg.Version = newPkgVersion.String()
	}
	return nil
}
