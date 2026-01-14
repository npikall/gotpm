/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"

	"github.com/npikall/gotpm/internal/system"
	"github.com/npikall/gotpm/internal/uninstall"
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
}

func uninstallRunner(cmd *cobra.Command, args []string) error {
	// Get System Environment
	goos := runtime.GOOS
	homeDir := Must(os.UserHomeDir())
	cwd := Must(os.Getwd())

	// Get Arguments
	var pkgName string
	if len(args) > 0 {
		pkgName = args[0]
	}

	// Attempt to open 'typst.toml'
	var tomlPkgName, tomlVersion string
	if pkg, err := system.OpenTypstTOML(cwd); err == nil {
		tomlPkgName = pkg.Name
		tomlVersion = pkg.Version
	}

	// Get Flag Values
	namespace := Must(cmd.Flags().GetString("namespace"))
	version := Must(cmd.Flags().GetString("version"))
	all := Must(cmd.Flags().GetBool("all"))
	isDryRun := Must(cmd.Flags().GetBool("dry-run"))

	// Overwrite pkgName if none in command and one in toml
	if pkgName == "" && tomlPkgName != "" {
		pkgName = tomlPkgName
	}

	// Overwrite version if none in command and one in toml
	// only when package name is not in command
	if version == "" && tomlVersion != "" && len(args) == 0 {
		version = tomlVersion
	}

	dataDir, err := system.GetTypstPath(goos, homeDir)
	target, err := uninstall.ResolveUninstallTarget(dataDir, all, namespace, pkgName, version)
	if err != nil {
		return err
	}
	if isDryRun {
		LogWarnf("perform dry-run")
	}

	isExisting, err := exists(target)
	if err != nil {
		return err
	}
	if !isExisting {
		return errors.New(fmt.Sprintf("path does not exist '%s'", target))
	}

	if isDryRun {
		LogInfof("deleting everything in '%s'", target)
		return nil
	}

	if err := os.RemoveAll(target); err != nil {
		return err
	}
	identifier := HighStyle.Render(fmt.Sprintf("@%s/%s:%s", namespace, pkgName, version))
	LogInfof("Uninstalled %s", identifier)
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
