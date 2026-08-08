/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/list"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all locally installed Packages",
	Long:  "List all locally installed Packages",
	Example: `# list all available Packages
gotpm list`,
	Args: cobra.NoArgs,
	RunE: ListRunner,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func ListRunner(cmd *cobra.Command, _ []string) error {
	return list.Run(newLogger(cmd))
}
