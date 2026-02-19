/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/npikall/gotpm/internal/files"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a Typst Package from the local Storage",
	Example: `gotpm uninstall # get name and version from typst.toml
gotpm uninstall foo
gotpm uninstall foo --namespace preview
gotpm uninstall foo --namespace preview --dry-run
`,
	RunE: uninstallRunner,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringP("namespace", "n", "local", "The namespace from which the package should be removed from.")
	uninstallCmd.Flags().StringP("version", "v", "", "The specific version of a package that should be removed.")
	uninstallCmd.Flags().Bool("all", false, "Uninstall all Packages from a given namespace or all versions of a package.")
	uninstallCmd.Flags().Bool("dry-run", false, "Perform a dry run.")
	uninstallCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
}

var ErrInsufficientPackage = errors.New("both package and version must be specified")

func uninstallRunner(cmd *cobra.Command, args []string) error {
	logger := setupVerboseLogger(cmd)
	cwd := Must(os.Getwd())

	var pkgName, pkgVersion string
	switch {
	case len(args) > 0:
		version := Must(cmd.Flags().GetString("version"))
		if version == "" {
			return ErrInsufficientPackage
		}
		pkgName = args[0]
		pkgVersion = version
		logger.Debug("from cli", "name", pkgName, "version", pkgVersion)
	default:
		pkg, err := files.LoadPackageFromDirectory(cwd)
		if err != nil {
			return err
		}
		pkgName = pkg.Name
		pkgVersion = pkg.Version
		logger.Debug("from toml", "name", pkgName, "version", pkgVersion)
	}

	// Get Flag Values
	namespace := Must(cmd.Flags().GetString("namespace"))
	deleteAll := Must(cmd.Flags().GetBool("all"))
	isDryRun := Must(cmd.Flags().GetBool("dry-run"))
	logger.Debug("run flags", "namespace", namespace, "all", deleteAll, "dry-run", isDryRun)

	typstPackagePath, err := paths.GetTypstPackagePath()
	if err != nil {
		return err
	}

	target, err := paths.ResolveUninstallTarget(typstPackagePath, deleteAll, namespace, pkgName, pkgVersion)
	if err != nil {
		return err
	}
	logger.Debug("uninstalling from", "path", target)

	if isDryRun {
		logger.Warn("perform dry-run")
	}

	isExisting := files.Exists(target)
	if !isExisting {
		logger.Errorf("path does not exist '%s'", target)
		return nil
	}

	if isDryRun {
		logger.Infof("deleting everything in '%s'", target)
		return nil
	}

	if err := os.RemoveAll(target); err != nil {
		return err
	}
	identifier := HighStyle.Render(fmt.Sprintf("@%s/%s:%s", namespace, pkgName, pkgVersion))
	logger.Infof("Uninstalled %s", identifier)
	return nil
}
