package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/remove"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove <@namespace/name:version>",
	Aliases: []string{"rm"},
	Short:   "Remove a dependency from this project.",
	Long: `Takes the import string exactly as it appears in typst.toml, so the
version being removed is never in doubt.

The dependency is dropped from typst.toml and from gotpm.lock, along with
any package only it pulled in. The installed files are left in the package
directory, because other projects on this machine may import the same version;
--prune deletes them too.
`,
	Example: `gotpm remove @gotpm/cetz:0.3.1
gotpm rm @gotpm/cetz:0.3.1 --prune
`,
	Args: cobra.ExactArgs(1),
	RunE: RemoveRunner,
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().Bool("prune", false, "Delete the removed packages from the package directory as well.")
}

func RemoveRunner(cmd *cobra.Command, args []string) error {
	opts := &remove.Options{
		Prune: Must(cmd.Flags().GetBool("prune")),
	}
	return remove.Run(args[0], opts, newLogger(cmd))
}
