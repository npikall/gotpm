/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
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
		logger := newLogger(cmd)
		target, err := paths.EnsureTypstPackagesDir()
		if err != nil {
			return fmt.Errorf("could not resolve package directory: %w", err)
		}
		logger.Debug("resolved", "path", target)
		ui.Infof("packages located at \"%s\"", ui.AccentBold.Render(target))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(locateCmd)
}
