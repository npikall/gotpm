package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/check"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check <file>",
	Short: "Check if all dependencies are available",
	Long:  `Check if all dependencies, that are imported by a file are available on the system`,
	Args:  cobra.ExactArgs(1),
	RunE:  CheckRunner,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func CheckRunner(cmd *cobra.Command, args []string) error {
	return check.Run(args[0], newLogger(cmd))
}
