/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/npikall/gotpm/internal/system"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "Locate the root directory, where the Typst Packages are stored.",
	Run: func(cmd *cobra.Command, args []string) {
		goos := runtime.GOOS
		homeDir := Must(os.UserHomeDir())
		dataDir := Must(system.GetDataDirectory(goos, homeDir))
		path := filepath.Join(dataDir, "typst", "packages")
		LogInfof("Packages located at: '%s'", path)
	},
}

func init() {
	rootCmd.AddCommand(locateCmd)
}
