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

	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a Typst Package from the local Storage",
	Long: `Removes a locally installed Typst package from the package directory.

The package directory can be overridden via the --install-dir flag
or the GOTPM_INSTALL_DIR environment variable. The flag takes precedence.
`,
	Example: `# get package metadata from typst.toml
gotpm uninstall
gotpm uninstall foo

# uninstall specific package from 'local' or 'preview'
gotpm uninstall foo -V 0.1.2
gotpm uninstall foo -V 0.1.2 -n preview

# all versions of foo in namespace 'local' or 'preview'
gotpm uninstall foo --all
gotpm uninstall foo -n preview --all

`,
	RunE: uninstallRunner,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringP("namespace", "n", "local", "The namespace from which the package should be removed from.")
	uninstallCmd.Flags().StringP("version", "V", "", "The specific version of a package that should be removed.")
	uninstallCmd.Flags().Bool("all", false, "Uninstall all Packages from a given namespace or all versions of a package.")
	uninstallCmd.Flags().Bool("dry-run", false, "Perform a dry run.")
	uninstallCmd.Flags().String(internal.InstallDirFlag, "", "Override the package directory (env: $"+internal.InstallDirEnvVar+")")
}

var ErrInsufficientPackage = errors.New("both package and version must be specified")

type UninstallFlags struct {
	Namespace string
	Version   string
	DeleteAll bool
	IsDryRun  bool
}

func uninstallRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)

	flags, err := ReadUninstallFlags(cmd)
	if err != nil {
		return err
	}
	logger.Debug("run flags", "namespace", flags.Namespace, "all", flags.DeleteAll, "dry-run", flags.IsDryRun)

	pkgName, pkgVersion, err := ResolvePackageIdentity(args, flags)
	if err != nil {
		return err
	}
	logger.Debug("resolved package", "name", pkgName, "version", pkgVersion)

	// Intentionally does not create the directory if it doesn't exist yet.
	installDir := internal.Must(cmd.Flags().GetString(internal.InstallDirFlag))
	localPkgDir, overridden, err := internal.ResolvePackageDirPath(installDir)
	if err != nil {
		return fmt.Errorf("could not resolve local package directory: %w", err)
	}
	logger.Debug("resolved local package directory", "path", localPkgDir)

	var target string
	if overridden {
		target = localPkgDir
	} else {
		target = ResolveUninstallTarget(localPkgDir, flags.Namespace, pkgName, pkgVersion, flags.DeleteAll)
	}
	logger.Debug("uninstalling from", "path", target)

	if err := ValidateTargetExists(target); err != nil {
		return err
	}

	if flags.IsDryRun {
		ui.Warnf("dryrun would delete %q", target)
		return nil
	}

	if err := RemoveTarget(target); err != nil {
		return err
	}
	importStmt := formatImportWithWildcard(flags, pkgVersion, pkgName)
	ui.Infof("uninstalled %s", importStmt)
	return nil
}

func formatImportWithWildcard(flags UninstallFlags, pkgVersion string, pkgName string) string {
	if flags.DeleteAll {
		pkgVersion = "*.*.*"
	}
	importStmt := ui.Import(flags.Namespace, pkgName, pkgVersion)
	return importStmt
}

func ResolvePackageIdentity(args []string, flags UninstallFlags) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("could not get current working directory: %w", err)
	}
	return ResolvePackageIdentityFromWorkingDir(args, flags.Version, flags.DeleteAll, cwd)
}

func ReadUninstallFlags(cmd *cobra.Command) (UninstallFlags, error) {
	deleteAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return UninstallFlags{}, fmt.Errorf("could not get flag '--all': %w", err)
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return UninstallFlags{}, fmt.Errorf("could not get flag '--version': %w", err)
	}
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return UninstallFlags{}, fmt.Errorf("could not get flag '--namespace': %w", err)
	}
	isDryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return UninstallFlags{}, fmt.Errorf("could not get flag '--dry-run': %w", err)
	}
	return UninstallFlags{
		Namespace: namespace,
		Version:   version,
		DeleteAll: deleteAll,
		IsDryRun:  isDryRun,
	}, nil
}

// ResolveUninstallTarget builds the path of the directory to remove.
// When deleteAll is true and no version is given, the package directory
// (all versions) is targeted; otherwise a specific version directory is used.
func ResolveUninstallTarget(pkgDir, namespace, name, version string, deleteAll bool) string {
	if deleteAll && version == "" {
		return filepath.Join(pkgDir, namespace, name)
	}
	return filepath.Join(pkgDir, namespace, name, version)
}

// ValidateTargetExists returns an error when there is nothing at target to remove.
// Uses Lstat so a dangling symlink still counts as "present".
func ValidateTargetExists(target string) error {
	if _, err := os.Lstat(target); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist %q: %w", target, err)
		}
		return fmt.Errorf("checking target %q: %w", target, err)
	}
	return nil
}

// RemoveTarget removes target from disk.
// When target is a symlink, only the link is removed, not the directory it points to.
// When target is a regular file or directory, it is removed with all its contents.
func RemoveTarget(target string) error {
	info, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("checking target %q: %w", target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("could not remove %q: %w", target, err)
		}
		return nil
	}
	if err = os.RemoveAll(target); err != nil {
		return fmt.Errorf("could not remove-all %q: %w", target, err)
	}
	return nil
}

// ResolvePackageIdentityFromWorkingDir returns the package name and version to uninstall.
// When a name is provided as an argument it is taken from CLI args; otherwise
// both are read from the typst.toml in dir.
func ResolvePackageIdentityFromWorkingDir(args []string, version string, deleteAll bool, dir string) (string, string, error) {
	if len(args) > 0 {
		if version == "" && !deleteAll {
			return "", "", ErrInsufficientPackage
		}
		return args[0], version, nil
	}
	manifest, err := internal.LoadManifest(dir)
	if err != nil {
		return "", "", fmt.Errorf("could not load typst manifest: %w", err)
	}
	return manifest.Package.Name, manifest.Package.Version, nil
}
