package cmd

import (
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal"
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
	RunE:  cacheClearRunner,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	cacheClearCmd.Flags().Bool("dry-run", false, "Dry run to see which data will be deleted")
}

func cacheClearRunner(cmd *cobra.Command, args []string) error {
	dataDir, err := internal.ResolveDataDir()
	if err != nil {
		return err
	}
	appDataDir := filepath.Join(dataDir, "gotpm")

	if isDryRun := internal.Must(cmd.Flags().GetBool("dry-run")); isDryRun {
		internal.PrintWarnf("dry-run, would clear cache at %q", appDataDir)
		return nil
	}

	if err = os.RemoveAll(appDataDir); err != nil {
		return err
	}

	internal.PrintInfof("Clear cache at %q", appDataDir)
	return nil
}
