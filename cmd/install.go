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
	"sync"

	"github.com/npikall/gotpm/internal/install"
	"github.com/npikall/gotpm/internal/system"
	"github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path] ",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied to
$DATA_DIR/typst/packages, where the $DATA_DIR is dependend on
the machines operating system.
`,
	Example: `gotpm install
gotpm install --editable
gotpm install --namespace preview
gotpm install path/to/package/dir
gotpm install path/to/package/dir -n preview
`,
	Run: installRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", "local", "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "If the installed package should be editable.")
}

func installRunner(cmd *cobra.Command, args []string) {
	goos := runtime.GOOS
	homeDir := Must(os.UserHomeDir())
	cwd := getCurrentWorkingDir(args)

	pkg := Must(system.OpenTypstTOML(cwd))
	namespace := Must(cmd.Flags().GetString("namespace"))
	isEditable := Must(cmd.Flags().GetBool("editable"))
	dst := Must(system.GetStoragePath(goos, homeDir, namespace, pkg.Name, pkg.Version))

	typstIgnorePath := filepath.Join(cwd, ".typstignore")
	typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
	// TODO: add exclude patterns from 'typst.toml'
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

		wg.Go(func() {
			processFile(path, targetPath, isEditable)
		})
		return nil
	})

	if err != nil {
		LogFatalf("%s", err)
	}
	wg.Wait()
	LogInfof("Package '%s' successfully installed", pkg.Name)
}

func processFile(srcPath, dstPath string, isEditable bool) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		LogErrf("%s", err)
		return
	}
	var err error
	switch isEditable {
	case true:
		err = os.Symlink(srcPath, dstPath)
	case false:
		err = install.CopyFile(srcPath, dstPath)
	}
	if err != nil {
		LogErrf("%s", err)
		return
	}
}

func getCurrentWorkingDir(args []string) string {
	if len(args) > 0 {
		cwd := Must(filepath.Abs(args[0]))
		return cwd
	}
	return Must(os.Getwd())
}
