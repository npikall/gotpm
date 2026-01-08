/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/npikall/gotpm/internal/echo"
	"github.com/npikall/gotpm/internal/system"
	"github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a Typst Package locally.",
	Run: func(cmd *cobra.Command, args []string) {
		goos, homeDir, err := system.GetSystemInfo()
		check(err)

		cwd, err := os.Getwd()
		check(err)
		pkg, err := system.OpenTypstTOML(cwd)
		if err != nil {
			echo.ExitErrorf("%s", err)
		}

		// TODO: make namespace changeable
		dst, err := system.GetStoragePath(goos, homeDir, "preview", pkg.Name, pkg.Version)
		if err != nil {
			echo.ExitErrorf("%s", err)
		}

		typstIgnorePath := filepath.Join(cwd, ".typstignore")
		typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
		if err != nil {
			echo.ExitErrorf("%s", err)
		}
		echo.EchoInfof("Installing to '%s'", dst)

		filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return err
			}
			switch {
			case strings.Contains(path, ".git"):
				return err
			case !typstIgnore.MatchesPath(path):
				echo.EchoInfof("found: %s", path)
				// TODO: add copy here
				return err
			default:
				return err
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
