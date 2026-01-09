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
	"runtime"
	"strings"

	"github.com/npikall/gotpm/internal/system"
	"github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a Typst Package locally.",
	Run: func(cmd *cobra.Command, args []string) {
		goos := runtime.GOOS
		homeDir := Must(os.UserHomeDir())
		cwd := Must(os.Getwd())

		// TODO: make namespace changeable
		pkg := Must(system.OpenTypstTOML(cwd))
		dst := Must(system.GetStoragePath(goos, homeDir, "preview", pkg.Name, pkg.Version))

		typstIgnorePath := filepath.Join(cwd, ".typstignore")
		typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
		if err != nil {
			typstIgnore = &ignore.GitIgnore{}
		}

		LogInfof("Installing to '%s'", dst)

		filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return err
			}
			switch {
			case strings.Contains(path, ".git"):
				return err
			case !typstIgnore.MatchesPath(path):
				LogInfof("found: %s", path)
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
}
