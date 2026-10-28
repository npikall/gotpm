package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/compiler"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// compilerCmd represents the compiler command
var compilerCmd = &cobra.Command{
	Use:   "compiler",
	Short: "Manage the compilers gotpm runs.",
	Long: `See and delete the compilers in gotpm's compiler directory, the ones
'gotpm run' runs, one directory per version. Like the font directory, it is
not cache: 'gotpm cache clear' leaves it alone.`,
}

var compilerListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List the installed compilers.",
	Long: `List the installed compilers, oldest first. The newest stable one is
marked, and so, inside a project, is the project's compiler pin, also when it
is not installed.`,
	Args: cobra.NoArgs,
	RunE: CompilerListRunner,
}

var compilerUninstallCmd = &cobra.Command{
	Use:   "uninstall <version>",
	Short: "Delete a compiler from the compiler directory.",
	Long: `Delete a compiler from the compiler directory, even when a project pins
it: the next 'gotpm run' in that project installs it again.`,
	Example: `gotpm compiler uninstall 0.13.1`,
	Args:    cobra.ExactArgs(1),
	RunE:    CompilerUninstallRunner,
}

func init() {
	rootCmd.AddCommand(compilerCmd)
	compilerCmd.AddCommand(compilerListCmd, compilerUninstallCmd)
}

func CompilerListRunner(cmd *cobra.Command, _ []string) error {
	return compiler.List(ui.Stdout(), newLogger(cmd))
}

func CompilerUninstallRunner(cmd *cobra.Command, args []string) error {
	return compiler.Uninstall(args[0], ui.Stdout(), newLogger(cmd))
}
