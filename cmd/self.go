package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/self"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// selfCmd represents the self command
var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Inspect or manage the gotpm binary",
}

var selfVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build information for the gotpm binary",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		ui.Print(asciiArt)
		self.Version(buildInfo())
	},
}

func init() {
	rootCmd.AddCommand(selfCmd)
	selfCmd.AddCommand(selfVersionCmd)
}

// buildInfo collects the values stamped into this package at build time.
func buildInfo() self.BuildInfo {
	return self.BuildInfo{
		Version:   gitTag,
		Commit:    gitCommit,
		OS:        buildOS,
		Arch:      buildARCH,
		Installer: installer,
	}
}
