package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [typst arguments...]",
	Short: "Run typst with a compiler gotpm installs.",
	Long: `Run typst, handing it every argument unchanged, the way 'typst' itself
would be run: 'gotpm run compile main.typ', 'gotpm run watch main.typ'.

The compiler comes from gotpm's compiler directory, installed first when
missing. A typst on $PATH is never used.

Inside a project, the compiler pin in typst.toml decides the compiler. When
the project pins none, or its pin is below the project's compiler requirement,
the newest installed stable compiler satisfying the requirement is pinned, or
else the latest stable release satisfying it, and run says so. Outside a
project, the newest installed stable compiler is used, or the latest stable
release when none is installed.

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
	return run.Run(cmd.Context(), args, ui.Stderr(), newLogger(cmd))
}
