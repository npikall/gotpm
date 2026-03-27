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

// selfCmd represents the self command
var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Inspect the gotpm binary in more detail",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gotpm version %s (%s) os=%s arch=%s", gitTag, gitCommit, buildOS, buildARCH)
	},
}

func init() {
	rootCmd.AddCommand(selfCmd)
}
