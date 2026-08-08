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
	Use:   "locate",
	Short: "Locate the root directory, where the Typst Packages are stored.",
	Long:  "Locate the root directory, where the Typst Packages are stored.",
	Example: `# Locate Typst Packages
gotpm locate`,
	Args: cobra.NoArgs,
	RunE: LocateRunner,
}

func init() {
	rootCmd.AddCommand(locateCmd)
}

func LocateRunner(cmd *cobra.Command, _ []string) error {
	return locate.Run(newLogger(cmd))
}
