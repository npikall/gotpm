/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"context"
	_ "embed"
	"os"

	"charm.land/fang/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/logger"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

//go:embed art.txt
var asciiArt string

var description string = `
GoTPM is a minimal Package Manager for Typst. Install the packages you write to
your disk, to make them installable via a local import.`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gotpm",
	Short: "A Package Manager for Typst written in Go.",
	Long:  asciiArt + description,
}

// installDirUsage is the help text of the --install-dir flag
const installDirUsage = "A directory holding one package's files, without namespace/name/version layout (env: $" +
	paths.InstallDirEnvVar + ")"

var (
	gitTag    string = "dev"
	gitCommit string = "00000000"
	buildOS   string = "NOOS"
	buildARCH string = "NOARCH"
	installer string = "source"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(gitTag),
		fang.WithCommit(gitCommit),
		fang.WithColorSchemeFunc(fang.AnsiColorScheme),
	); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().CountP("verbose", "v", "enable verbose output")
	rootCmd.Flags().BoolP("version", "V", false, "version for gotpm")
}

func newLogger(cmd *cobra.Command) *log.Logger {
	count, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		count = 0
	}
	return logger.Setup(count)
}

func Must[T any](t T, err error) T { //nolint: ireturn
	if err != nil {
		ui.Error(err)
		os.Exit(1)
	}
	return t
}
