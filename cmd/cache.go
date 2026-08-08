package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/cache"
	"github.com/spf13/cobra"
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the cache of gotpm",
	Long:  `Manage the cache of gotpm`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the Cache",
	Args:  cobra.NoArgs,
	RunE:  CacheClearRunner,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	cacheClearCmd.Flags().Bool("dry-run", false, "Dry run to see which data will be deleted")
}

func CacheClearRunner(cmd *cobra.Command, _ []string) error {
	opts := &cache.Options{
		DryRun: Must(cmd.Flags().GetBool("dry-run")),
	}
	return cache.Clear(opts, newLogger(cmd))
}
