/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/uninstall"
	"github.com/npikall/gotpm/internal/paths"
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
	Args: cobra.MaximumNArgs(1),
	RunE: UninstallRunner,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "The namespace from which the package should be removed from.")
	uninstallCmd.Flags().StringP("version", "V", "", "The specific version of a package that should be removed.")
	uninstallCmd.Flags().Bool("all", false, "Uninstall all Packages from a given namespace or all versions of a package.")
	uninstallCmd.Flags().Bool("dry-run", false, "Perform a dry run.")
	uninstallCmd.Flags().String(paths.InstallDirFlag, "", "Override the package directory (env: $"+paths.InstallDirEnvVar+")")
}

func UninstallRunner(cmd *cobra.Command, args []string) error {
	opts := &uninstall.Options{
		Namespace:  Must(cmd.Flags().GetString("namespace")),
		Version:    Must(cmd.Flags().GetString("version")),
		All:        Must(cmd.Flags().GetBool("all")),
		DryRun:     Must(cmd.Flags().GetBool("dry-run")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	return uninstall.Run(name, opts, newLogger(cmd))
}
