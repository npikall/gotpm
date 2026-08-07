/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/remove"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove <@gotpm/name:version>",
	Aliases: []string{"rm"},
	Short:   "Remove a dependency from this project.",
	Long: `Takes the import string exactly as it appears in typst.toml, so the
version being removed is never in doubt.

The dependency is dropped from typst.toml and from gotpm.lock, along with
any package only it pulled in. The installed files are left in the package
store, because other projects on this machine may import the same version;
--prune deletes them too.
`,
	Example: `gotpm remove @gotpm/cetz:0.3.1
gotpm rm @gotpm/cetz:0.3.1 --prune
`,
	Args: cobra.ExactArgs(1),
	RunE: RemoveRunner,
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().Bool("prune", false, "Delete the removed packages from the package store as well.")
	removeCmd.Flags().String(paths.InstallDirFlag, "", "Override the package directory (env: $"+paths.InstallDirEnvVar+")")
}

func RemoveRunner(cmd *cobra.Command, args []string) error {
	opts := &remove.Options{
		Prune:      Must(cmd.Flags().GetBool("prune")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
	}
	return remove.Run(args[0], opts, newLogger(cmd))
}
