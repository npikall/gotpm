package cmd

import (
	"fmt"
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

	dirSize := DirSizeMB(appDataDir)

	if isDryRun := internal.Must(cmd.Flags().GetBool("dry-run")); isDryRun {
		internal.PrintWarnf("dry-run, would clear %s of cached data at %q", dirSize, appDataDir)
		return nil
	}

	if err = os.RemoveAll(appDataDir); err != nil {
		return err
	}

	internal.PrintInfof("Clear %s cached data at %q", dirSize, appDataDir)
	return nil
}

func DirSizeMB(path string) string {
	size, err := DirSize(path)
	if err != nil {
		return "?"
	}
	sizeMB := float64(size) / 1024.0 / 1024.0
	return fmt.Sprintf("%.1fMB", sizeMB)
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
