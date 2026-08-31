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
gotpm init mypkg`,
	Short: "Initialize a new minimal Typst Package",
	Args:  cobra.MaximumNArgs(1),
	RunE:  InitRunner,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("doc", false, "Initialize a Document Manifest")
	initCmd.Flags().Bool("lib", false, "Initialize a Library Manifest")
}

func InitRunner(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	opts := scaffold.Options{
		Document: Must(cmd.Flags().GetBool("doc")),
		Library:  Must(cmd.Flags().GetBool("lib")),
	}
	return scaffold.Run(name, opts, newLogger(cmd))
}
