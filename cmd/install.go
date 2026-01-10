/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/npikall/gotpm/internal/install"
	"github.com/npikall/gotpm/internal/system"
	"github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a Typst Package locally.",
	Run:   installRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", "local", "The namespace in which the package should be available.")
}

func installRunner(cmd *cobra.Command, args []string) {
	goos := runtime.GOOS
	homeDir := Must(os.UserHomeDir())
	cwd := Must(os.Getwd())

	pkg := Must(system.OpenTypstTOML(cwd))
	namespace := Must(cmd.Flags().GetString("namespace"))
	dst := Must(system.GetStoragePath(goos, homeDir, namespace, pkg.Name, pkg.Version))

	typstIgnorePath := filepath.Join(cwd, ".typstignore")
	typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
	if err != nil {
		typstIgnore = &ignore.GitIgnore{}
		LogWarnf("No '.typstignore' file. Copy all in '%s'", cwd)
	}

	LogInfof("Installing to '%s'", dst)

	var wg sync.WaitGroup

	err = filepath.WalkDir(cwd, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Ignore Directories and the .git folder
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		if typstIgnore.MatchesPath(path) {
			return nil
		}

		targetPath := Must(install.ResolveTargetPath(cwd, path, dst))
		worker := install.CopyWorker{Src: path, Dst: targetPath}
		wg.Go(worker.Copy)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()
	LogInfof("Package '%s' successfully installed", pkg.Name)
}
