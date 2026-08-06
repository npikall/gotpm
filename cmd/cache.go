package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/ui"
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
	RunE:  CacheClearRunner,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	cacheClearCmd.Flags().Bool("dry-run", false, "Dry run to see which data will be deleted")
}

func CacheClearRunner(cmd *cobra.Command, args []string) error {
	remotesDir, err := internal.ResolveRemotesDir()
	if err != nil {
		return err
	}
	cachePath, err := internal.ResolveCachePath()
	if err != nil {
		return err
	}

	size := cacheClearSizeMB(remotesDir, cachePath)

	if isDryRun := Must(cmd.Flags().GetBool("dry-run")); isDryRun {
		ui.Warnf("dry-run, would clear %s (remotes: %q, index cache: %q)", size, remotesDir, cachePath)
		return nil
	}

	if err = os.RemoveAll(remotesDir); err != nil {
		return err //nolint: wrapcheck
	}
	if err = os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return err //nolint: wrapcheck
	}

	ui.Infof("Cleared %s (remotes: %q, index cache: %q)", size, remotesDir, cachePath)
	return nil
}

func cacheClearSizeMB(remotesDir, cachePath string) string {
	size, err := DirSize(remotesDir)
	if err != nil {
		return "?"
	}
	if info, err := os.Stat(cachePath); err == nil {
		size += info.Size()
	}
	sizeMB := float64(size) / 1024.0 / 1024.0 //nolint: mnd
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
	return size, err //nolint: wrapcheck
}
