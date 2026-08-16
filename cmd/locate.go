/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/locate"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate [key]",
	Short: "Show the paths and directories gotpm reads and writes.",
	Long: `Show the paths and directories gotpm reads and writes.

Without a key, every path is listed, grouped by what it belongs to. The
project group is only shown when the working directory belongs to a typst
project.

With a key, only that path is printed, unstyled and on its own, so it can be
used directly in a shell.

The packages path answers to $GOTPM_INSTALL_DIR first and $TYPST_PACKAGE_PATH
second, and the note beside it says which one applied. They are not the same
kind of path: $TYPST_PACKAGE_PATH moves the package directory typst imports
from, keeping its namespace/name/version layout, while $GOTPM_INSTALL_DIR names
a directory that receives one package's files directly, without that layout.

Nothing is created: a path that does not exist yet is still where gotpm would
look for it.`,
	Example: `# Show every path
gotpm locate

# Print one path, for use in a shell
cd "$(gotpm locate packages)"`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: locate.Keys(),
	RunE:      LocateRunner,
}

func init() {
	rootCmd.AddCommand(locateCmd)
}

func LocateRunner(cmd *cobra.Command, args []string) error {
	var key string
	if len(args) > 0 {
		key = args[0]
	}
	return locate.Run(key, newLogger(cmd))
}
