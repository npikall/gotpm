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
	"sync"

	"github.com/charmbracelet/log"
	"github.com/npikall/gotpm/internal/files"
	"github.com/npikall/gotpm/internal/paths"
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
	RunE: installRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", "local", "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "If the installed package should be editable.")
	installCmd.Flags().BoolP("verbose", "V", false, "Print Debug Level Information")
}

func installRunner(cmd *cobra.Command, args []string) error {
	logger := setupVerboseLogger(cmd)

	cwd := getCurrentWorkingDir(args)
	logger.Debug("running in", "cwd", cwd)

	pkg, err := files.LoadPackageFromDirectory(cwd)
	if err != nil {
		return err
	}

	namespace := Must(cmd.Flags().GetString("namespace"))
	isEditable := Must(cmd.Flags().GetBool("editable"))
	logger.Debug("flag", "namespace", namespace)
	logger.Debug("flag", "editable", isEditable)
	typstPackagePath, err := paths.GetTypstPackagePath()
	if err != nil {
		return err
	}
	target := filepath.Join(typstPackagePath, namespace, pkg.Name, pkg.Version)

	typstIgnorePath := filepath.Join(cwd, ".typstignore")
	typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
	// TODO: add exclude patterns from 'typst.toml'
	if err != nil {
		typstIgnore = &ignore.GitIgnore{}
		logger.Warnf("No '.typstignore' file. Copy all in '%s'", cwd)
	}

	logger.Infof("Installing to '%s'", target)

	var wg sync.WaitGroup
	logger.Debug("start walking", "dir", cwd)
	err = filepath.WalkDir(cwd, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Ignore Directories and the .git folder
		if d.IsDir() {
			if d.Name() == ".git" {
				logger.Debug("skip dir .git/")
				return fs.SkipDir
			}
			logger.Debug("skip dir", "dir", d.Name())
			return nil
		}

		if typstIgnore.MatchesPath(path) {
			logger.Debug("ignore matches", "path", path)
			return nil
		}

		targetPath := Must(paths.ResolveTargetPath(cwd, path, target))
		logger.Debug("resolved", "targetPath", targetPath)

		wg.Go(func() {
			processFile(logger, path, targetPath, isEditable)
		})
		return nil
	})

	if err != nil {
		return err
	}

	wg.Wait()
	logger.Infof("package '%s' successfully installed", pkg.Name)
	return nil
}

func processFile(logger *log.Logger, srcPath, dstPath string, isEditable bool) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		logger.Error(err)
		return
	}
	var err error
	switch isEditable {
	case true:
		err = os.Symlink(srcPath, dstPath)
	case false:
		err = files.CopyFile(srcPath, dstPath)
	}
	if err != nil {
		logger.Error(err)
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
