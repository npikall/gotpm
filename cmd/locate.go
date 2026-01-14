/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"os"
	"runtime"

	"github.com/npikall/gotpm/internal/system"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:     "locate",
	Short:   "Locate the root directory, where the Typst Packages are stored.",
	Example: `gotpm locate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		goos := runtime.GOOS
		homeDir := Must(os.UserHomeDir())
		path, err := system.GetTypstPath(goos, homeDir)
		if err != nil {
			return err
		}
		LogInfof("Packages located at: '%s'", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(locateCmd)
}
