package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/publish"
	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a Package to the Typst Universe.",
	Long: `Publish a Typst Package to the Typst Universe.
This involves pushing your changes to a fork of the github.com/typst/packages repo
on a dedicated branch, ready for you to open a Pull Request from.

GoTPM will know where your fork lives on disc, and handle committing your
Package files to the correct location.`,
	Example: `gotpm publish
gotpm publish --local`,
	Args: cobra.NoArgs,
	RunE: PublishRunner,
}

func init() {
	rootCmd.AddCommand(publishCmd)
	publishCmd.Flags().Bool("local", false, "Stop after committing to the local fork clone; do not push.")
}

func PublishRunner(cmd *cobra.Command, _ []string) error {
	opts := &publish.Options{
		Local: Must(cmd.Flags().GetBool("local")),
	}
	return publish.Run(opts, newLogger(cmd))
}
