/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/scaffold"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use: "init",
	Example: `# initialize a new Package
gotpm init`,
	Short: "Initialize a new minimal Typst Package",
	Args:  cobra.MaximumNArgs(1),
	RunE:  InitRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func InitRunner(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	return scaffold.Run(name, newLogger(cmd))
}
