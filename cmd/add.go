/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/add"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <repository>",
	Short: "Add a repository as a dependency of this project.",
	Long: `The package is installed under the @gotpm namespace, together with
everything it depends on, and recorded in two files next to typst.toml:

  typst.toml   gains the import under [tool.gotpm].dependencies
  gotpm.lock   pins every package to the exact commit it was fetched from

Commit both. gotpm.lock is what lets anyone who checks the project out run
'gotpm sync' and get the same packages, and it is the only place recording
where a dependency's own dependencies come from.

Without --rev the newest release tag is used, or the current HEAD when the
repository has no release tags.
`,
	Example: `gotpm add github.com/user/repo
gotpm add github.com/user/repo -t v0.1.2
gotpm add git@github.com:user/repo.git
`,
	Args: cobra.ExactArgs(1),
	RunE: AddRunner,
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("rev", "t", "", "The revision (hash or tag) to pin. Defaults to the newest release.")
	addCmd.Flags().BoolP("force", "f", false, "Replace a package installed from a different repository.")
	addCmd.Flags().String(paths.InstallDirFlag, "", installDirUsage)
}

func AddRunner(cmd *cobra.Command, args []string) error {
	opts := &add.Options{
		Revision:   Must(cmd.Flags().GetString("rev")),
		Force:      Must(cmd.Flags().GetBool("force")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
	}
	return add.Run(args[0], opts, newLogger(cmd))
}
