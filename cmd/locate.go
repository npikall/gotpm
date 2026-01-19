/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:     "locate",
	Short:   "Locate the root directory, where the Typst Packages are stored.",
	Example: `gotpm locate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := setupLogger()
		target, err := paths.GetTypstPackagePath()
		if err != nil {
			return err
		}
		logger.Info("packages at", "path", target)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(locateCmd)
}
