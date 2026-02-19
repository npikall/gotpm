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
	pkg, err := files.LoadPackageFromDirectory(cwd)
	if err != nil {
		return err
	}
	logger.Debug("running in", "cwd", cwd)

	isShow := Must(cmd.Flags().GetBool("show-current"))
	if isShow {
		fmt.Println(pkg.Version)
		return nil
	}

	oldPkgVersion := pkg.Version
	newPkgVersion, err := version.ParseVersion(oldPkgVersion)
	logger.Debug("from 'typst.toml'", "version", oldPkgVersion)
	if err != nil {
		return err
	}

	// Handle Runtime Flags
	var bump string
	if len(args) > 0 {
		bump = args[0]
	} else {
		return ErrMissingArgument
	}
	dryRun := Must(cmd.Flags().GetBool("dry-run"))

	isFixedVersion := version.IsSemVer(bump)

	switch {
	case isFixedVersion:
		pkg.Version = bump
		logger.Debug("setting toml", "version", bump)
	default:
		err := newPkgVersion.Bump(bump)
		if err != nil {
			return err
		}
		pkg.Version = newPkgVersion.String()
		logger.Debug("setting toml", "version", newPkgVersion.String())
	}

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

	// Read the existing TOML file
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
