/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	selfVersion string = "dev"
	selfName    string = "gotpm"
	selfCommit  string = "unknown"
)

// selfCmd represents the self command
var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Inspect the binary itself.",
	Long:  `Print the version and build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		isFull := Must(cmd.Flags().GetBool("full"))
		if isFull {
			fmt.Printf("%s %s at %s\n", selfName, selfVersion, selfCommit)
		} else {
			fmt.Printf("%s %s\n", selfName, selfVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(selfCmd)
	selfCmd.Flags().BoolP("full", "f", false, "Full description of the build version")
}
