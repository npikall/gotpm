/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/sync"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Install everything this project depends on.",
	Long: `Reads typst.toml and gotpm.lock and makes the package store match them.
This is what a fresh checkout needs before it compiles.

Every package is installed at the commit gotpm.lock pins, not at whatever
its tag points at today. Lock entries that nothing in typst.toml requires
any more are dropped.

--frozen fails instead of rewriting gotpm.lock, which is what a CI job
wants: a lock that disagrees with typst.toml is a change somebody forgot
to commit.
`,
	Example: `gotpm sync
gotpm sync --frozen
`,
	Args: cobra.NoArgs,
	RunE: SyncRunner,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().Bool("frozen", false, "Fail instead of updating gotpm.lock.")
	syncCmd.Flags().BoolP("force", "f", false, "Replace a package installed from a different repository.")
	syncCmd.Flags().String(paths.InstallDirFlag, "", "Override the package directory (env: $"+paths.InstallDirEnvVar+")")
}

func SyncRunner(cmd *cobra.Command, _ []string) error {
	opts := &sync.Options{
		Frozen:     Must(cmd.Flags().GetBool("frozen")),
		Force:      Must(cmd.Flags().GetBool("force")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
	}
	return sync.Run(opts, newLogger(cmd))
}
