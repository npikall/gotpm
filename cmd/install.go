/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/install"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied to
$DATA_DIR/typst/packages, where the $DATA_DIR is dependent on
the machine's operating system.

The destination directory can be overridden via the --install-dir flag
or the GOTPM_INSTALL_DIR environment variable. The flag takes precedence.
`,
	Example: `gotpm install
gotpm install . -e
gotpm install -n preview
gotpm install -r github.com/user/repo -t v0.1.2
gotpm install path/to/package -n preview
`,
	Args: cobra.MaximumNArgs(1),
	RunE: InstallRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "Create a symlink to the source directory instead of copying files.")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite an already-installed package.")
	installCmd.Flags().String(paths.InstallDirFlag, "", "Override the package directory (env: $"+paths.InstallDirEnvVar+")")
	installCmd.Flags().StringP("remote", "r", "", "The remote repository which should be installed.")
	installCmd.Flags().StringP("rev", "t", "HEAD", "The revision (hash or tag) that should be checked out.")
}

func InstallRunner(cmd *cobra.Command, args []string) error {
	opts := &install.Options{
		Force:      Must(cmd.Flags().GetBool("force")),
		Editable:   Must(cmd.Flags().GetBool("editable")),
		Namespace:  Must(cmd.Flags().GetString("namespace")),
		Remote:     Must(cmd.Flags().GetString("remote")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
		Revision:   Must(cmd.Flags().GetString("rev")),
	}

	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	return install.Run(path, opts, newLogger(cmd))
}
