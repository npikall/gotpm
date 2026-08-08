/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/self"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gotpm to the latest version from GitHub Releases",
	Long:  "Download and install the latest gotpm release from GitHub, replacing the current binary in place.",
	Args:  cobra.NoArgs,
	RunE:  SelfUpdateRunner,
}

func init() {
	selfCmd.AddCommand(selfUpdateCmd)
	selfUpdateCmd.Flags().Bool("check", false, "check for an update without installing it")
}

func SelfUpdateRunner(cmd *cobra.Command, _ []string) error {
	opts := &self.Options{
		CheckOnly: Must(cmd.Flags().GetBool("check")),
	}
	return self.Update(buildInfo(), opts)
}
