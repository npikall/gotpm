/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "Locate the root directory, where the Typst Packages are stored.",
	Long:  "Locate the root directory, where the Typst Packages are stored.",
	Example: `# Locate Typst Packages
gotpm locate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := internal.SetupLogger(cmd)
		target, err := internal.ResolveLocalPackageDir()
		if err != nil {
			return err
		}
		logger.Debug("resolved", "path", target)
		internal.PrintInfo("%s %q", internal.StyleMuted.Render("packages located at"), target)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(locateCmd)
}
