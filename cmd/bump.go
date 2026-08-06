/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"

	"github.com/npikall/gotpm/internal/cmds/bump"
	"github.com/spf13/cobra"
)

// bumpCmd represents the version command
var bumpCmd = &cobra.Command{
	Use: "bump [increment|version]",
	Example: `# bump with a given increment
gotpm bump major

# set to a specific version
gotpm bump 0.1.2
`,
	Short: "Manage the version of a Typst Package",
	Long: `Use this command to change the version of the Package or to display it.
Valid arguments can be:
	- major
	- minor
	- patch
	- a valid semantic version (e.g. 0.1.2)`,
	RunE: BumpRunner,
	// --show-current needs no increment, everything else does; the check
	// lives in BumpRunner rather than in Args.
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(bumpCmd)
	bumpCmd.Flags().Bool("dry-run", false, "Perform a dry-run")
	bumpCmd.Flags().BoolP("show-current", "c", false, "Show the version of the current package")
	bumpCmd.Flags().BoolP("show-next", "n", false, "Show the version of the package if it where bumped")
	bumpCmd.Flags().BoolP("indent", "i", false, "Use Indentation in the typst.toml file.")
}

var ErrMissingArgument = errors.New("argument must be provided, can be one of [major|minor|patch] or a valid semver")

func BumpRunner(cmd *cobra.Command, args []string) error {
	opts := &bump.Options{
		DryRun:   Must(cmd.Flags().GetBool("dry-run")),
		ShowCur:  Must(cmd.Flags().GetBool("show-current")),
		ShowNext: Must(cmd.Flags().GetBool("show-next")),
		Indent:   Must(cmd.Flags().GetBool("indent")),
	}

	increment := ""
	if len(args) > 0 {
		increment = args[0]
	} else if !opts.ShowCur {
		return ErrMissingArgument
	}
	return bump.Run(increment, opts, newLogger(cmd))
}
