package cmd

import (
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
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(bumpCmd)
	bumpCmd.Flags().Bool("dry-run", false, "Perform a dry-run")
	bumpCmd.Flags().BoolP("show-current", "c", false, "Show the version of the current package")
	bumpCmd.Flags().BoolP("show-next", "n", false, "Show the version of the package if it where bumped")
	bumpCmd.Flags().BoolP("indent", "i", false, "Use Indentation in the typst.toml file.")
}

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
	}
	return bump.Run(increment, opts, newLogger(cmd))
}
