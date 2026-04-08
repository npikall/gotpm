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
	"path/filepath"

	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a Typst Package from the local Storage",
	Example: `# get package metadata from typst.toml
gotpm uninstall
gotpm uninstall foo

# uninstall specific package from 'local' or 'preview'
gotpm uninstall foo -v 0.1.2
gotpm uninstall foo -v 0.1.2 -n preview

# all versions of foo in namespace 'local' or 'preview'
gotpm uninstall foo --all
gotpm uninstall foo -n preview --all

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

var ErrInsufficientPackage = errors.New("both package and version must be specified")

type uninstallFlags struct {
	namespace string
	version   string
	deleteAll bool
	isDryRun  bool
}

func uninstallRunner(cmd *cobra.Command, args []string) error {
	logger := setupLogger(cmd)

	flags, err := readUninstallFlags(cmd)
	if err != nil {
		return err
	}
	logger.Debug("run flags", "namespace", flags.namespace, "all", flags.deleteAll, "dry-run", flags.isDryRun)

	pkgName, pkgVersion, err := resolvePackageIdentity(args, flags)
	if err != nil {
		return err
	}
	logger.Debug("resolved package", "name", pkgName, "version", pkgVersion)

	// Intentionally use resolveLocalPackageDirPath (not resolveLocalPackageDir):
	// uninstall must not create the packages directory if it doesn't exist yet.
	localPkgDir, err := resolveLocalPackageDirPath()
	if err != nil {
		return err
	}
	logger.Debug("resolved local package directory", "path", localPkgDir)

	target := resolveUninstallTarget(localPkgDir, flags.namespace, pkgName, pkgVersion, flags.deleteAll)
	logger.Debug("uninstalling from", "path", target)

	if err := validateTargetExists(target); err != nil {
		return err
	}

	if flags.isDryRun {
		printWarn("dry-run: would delete %q", target)
		return nil
	}

	if err := removeTarget(target); err != nil {
		return err
	}
	printInfo("uninstalled %s", formatImportStmt(flags.namespace, pkgName, pkgVersion))
	return nil
}

func resolvePackageIdentity(args []string, flags uninstallFlags) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	pkgName, pkgVersion, err := resolvePackageIdentityFromWorkingDir(args, flags.version, flags.deleteAll, cwd)
	if err != nil {
		return "", "", err
	}
	return pkgName, pkgVersion, nil
}

func readUninstallFlags(cmd *cobra.Command) (uninstallFlags, error) {
	deleteAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return uninstallFlags{}, err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return uninstallFlags{}, err
	}
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return uninstallFlags{}, err
	}
	isDryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return uninstallFlags{}, err
	}
	return uninstallFlags{
		namespace: namespace,
		version:   version,
		deleteAll: deleteAll,
		isDryRun:  isDryRun,
	}, nil
}

// Build the path of the directory to remove.
// When deleteAll is true and no version is given, the package directory
// (all versions) is targeted; otherwise a specific version directory is used.
func resolveUninstallTarget(pkgDir, namespace, name, version string, deleteAll bool) string {
	if deleteAll && version == "" {
		return filepath.Join(pkgDir, namespace, name)
	}
	return filepath.Join(pkgDir, namespace, name, version)
}

// Return an error when there is nothing at target to remove.
// Uses Lstat so a dangling symlink still counts as "present".
func validateTargetExists(target string) error {
	if _, err := os.Lstat(target); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist %q", target)
		}
		return fmt.Errorf("checking target %q: %w", target, err)
	}
	return nil
}

// Remove target from disk.
// When target is a symlink, only the link is removed, not the directory it points to.
// When target is a regular file or directory, it is removed with all its contents.
func removeTarget(target string) error {
	info, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("checking target %q: %w", target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(target)
	}
	return os.RemoveAll(target)
}

// Return the package name and version to uninstall.
// When a name is provided as an argument it is taken from CLI flags; otherwise
// both are read from the typst.toml in dir.
func resolvePackageIdentityFromWorkingDir(args []string, version string, deleteAll bool, dir string) (name, ver string, err error) {
	if len(args) > 0 {
		if version == "" && !deleteAll {
			return "", "", ErrInsufficientPackage
		}
		return args[0], version, nil
	}
	manifest, err := loadManifest(dir)
	if err != nil {
		return "", "", err
	}
	return manifest.Package.Name, manifest.Package.Version, nil
}
