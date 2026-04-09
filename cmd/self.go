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

// selfCmd represents the self command
var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Inspect the gotpm binary in more detail",
	Run: func(cmd *cobra.Command, args []string) {
		internal.PrintInfo("gotpm version=%s hash=%s os=%s arch=%s\n", gitTag, gitCommit, buildOS, buildARCH)
	},
}

func init() {
	rootCmd.AddCommand(selfCmd)
}
