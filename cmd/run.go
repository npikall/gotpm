package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [typst arguments...]",
	Short: "Run typst with a compiler gotpm installs.",
	Long: `Run typst, handing it every argument unchanged, the way 'typst' itself
would be run: 'gotpm run compile main.typ', 'gotpm run watch main.typ'.

The compiler comes from gotpm's compiler directory. The newest installed
stable compiler is used; when none is installed, the latest stable release is
installed first. A typst on $PATH is never used.

gotpm's own flags end at the first argument that is not one of them, so
'gotpm run compile --help' shows typst's help. Put '--' first to hand typst a
flag of its own, as in 'gotpm run -- --version'.`,
	Example: `gotpm run compile main.typ
gotpm run watch main.typ
gotpm run -- --version`,
	Args: cobra.ArbitraryArgs,
	RunE: RunRunner,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().SetInterspersed(false)
}

func RunRunner(cmd *cobra.Command, args []string) error {
	return run.Run(cmd.Context(), args, newLogger(cmd))
}
