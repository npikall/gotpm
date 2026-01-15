/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	version "github.com/npikall/gotpm/internal/bump"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/system"
	"github.com/spf13/cobra"
)

// bumpCmd represents the version command
var bumpCmd = &cobra.Command{
	Use: "bump",
	Example: ` gotpm bump major
gotpm bump 0.1.2
`,
	Short: "Manage the version of a Typst Package",
	Long:  `Use this command to change the version of the Package or to display it.`,
	RunE:  bumpRunner,
}

const bumpDefault string = "none"

func init() {
	rootCmd.AddCommand(bumpCmd)
	bumpCmd.Flags().Bool("dry-run", false, "Perform a dry-run")
	bumpCmd.Flags().BoolP("verbose", "V", false, "Print Debug Level Information")
}

var ErrMissingArgument = errors.New("argument must be provided, can be one of [major|minor|patch] or a valid semver")

func bumpRunner(cmd *cobra.Command, args []string) error {
	verbose := Must(cmd.Flags().GetBool("verbose"))
	logger := setupLogger(verbose)

	cwd := Must(os.Getwd())
	pkg, err := system.OpenTypstTOML(cwd)
	if err != nil {
		return err
	}
	logger.Debugf("running in %s", cwd)

	oldPkgVersion := pkg.Version
	newPkgVersion, err := version.ParseVersion(oldPkgVersion)
	logger.Debugf("Version from 'typst.toml' %s", oldPkgVersion)
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
		logger.Debug("setting toml version to", "bump", bump)
	default:
		err := newPkgVersion.Bump(bump)
		if err != nil {
			return err
		}
		pkg.Version = newPkgVersion.String()
		logger.Debug("setting toml version to", "bump", newPkgVersion.String())
	}

	if dryRun {
		logger.Warn("performing dry-run")
		logger.Infof("updated version %s -> %s", oldPkgVersion, pkg.Version)
		return nil
	}

	// Read the existing TOML file
	typstTOML := filepath.Join(cwd, "typst.toml")
	typstTOMLContent, err := os.ReadFile(typstTOML)
	if err != nil {
		return err
	}
	logger.Debugf("editing file %s", typstTOML)

	// Write updated TOML to a buffer first
	var buf bytes.Buffer
	err = manifest.WriteTOML(&buf, pkg, typstTOMLContent)
	if err != nil {
		return err
	}
	logger.Debug("write edited toml to buffer")

	// Write the buffer to the file
	err = os.WriteFile(typstTOML, buf.Bytes(), 0644)
	if err != nil {
		return err
	}
	logger.Debugf("write buffer to file %s", typstTOML)

	logger.Infof("updated version %s -> %s", oldPkgVersion, pkg.Version)
	return nil
}
