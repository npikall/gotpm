package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/scaffold"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use: "init [name]",
	Example: `# initialize a new Package
gotpm init

# scaffold into a new directory
gotpm init mypkg

# scaffold a document project, which uses packages instead of being one
gotpm init thesis --doc`,
	Short: "Initialize a new minimal Typst package or document project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  InitRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
	addInitFlags(initCmd)
}

// addInitFlags registers the flags naming what init scaffolds. A project is
// either a package or a document, so the two are mutually exclusive.
func addInitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("doc", false, "Scaffold a document project")
	cmd.Flags().Bool("pkg", false, "Scaffold a package (default)")
	cmd.MarkFlagsMutuallyExclusive("doc", "pkg")
}

func InitRunner(cmd *cobra.Command, args []string) error {
	opts := scaffold.Options{Kind: kindFromFlags(cmd)}
	return scaffold.Run(firstArg(args), opts, newLogger(cmd))
}

// kindFromFlags reads what init scaffolds off its flags. --pkg needs no branch
// of its own: a package is what an unset kind means.
func kindFromFlags(cmd *cobra.Command) scaffold.Kind {
	if Must(cmd.Flags().GetBool("doc")) {
		return scaffold.KindDocument
	}
	return scaffold.KindPackage
}
