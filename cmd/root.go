/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var asciiArt string = LogoStyle.Render(`┌──────────────────────────────┐
│ _____     ______________  ___│
│|  __ \   |_   _| ___ \  \/  |│
│| |  \/ ___ | | | |_/ / .  . |│
│| | __ / _ \| | |  __/| |\/| |│
│| |_\ \ (_) | | | |   | |  | |│
│ \____/\___/\_/ \_|   \_|  |_/│
└──────────────────────────────┘
GoTPM is a minimal Package Manager for Typst.
Install the packages you write to your disk, to
make them installable via a local import.`)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "gotpm [command] [--flags]",
	Short:   "A Package Manager for Typst written in Go.",
	Long:    asciiArt,
	Version: GoTPMVersion,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}

func init() {
}
