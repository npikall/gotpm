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

	"github.com/npikall/gotpm/internal/files"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied to
$DATA_DIR/typst/packages, where the $DATA_DIR is dependend on
the machines operating system.
`,
	Example: `# install Package located in the CWD
gotpm install
gotpm install --editable
gotpm install --namespace preview

# install a Package not in the CWD
gotpm install path/to/package/dir
gotpm install path/to/package/dir -n preview
`,
	RunE: installRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", "local", "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "If the installed package should be editable.")
	installCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
	installCmd.Flags().Bool("dry-run", false, "Perform a dry-run")
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
	logger.Debug("flag", "namespace", namespace)

	isEditable := Must(cmd.Flags().GetBool("editable"))
	logger.Debug("flag", "editable", isEditable)

	isDryRun := Must(cmd.Flags().GetBool("dry-run"))
	logger.Debug("flag", "dry-run", isDryRun)

	typstPackagePath, err := paths.GetTypstPackagePath()
	if err != nil {
		return err
	}

	dstDir := filepath.Join(typstPackagePath, namespace, pkg.Name, pkg.Version)

	typstIgnorePath := filepath.Join(cwd, ".typstignore")
	typstIgnore, err := ignore.CompileIgnoreFile(typstIgnorePath)
	// TODO: add exclude patterns from 'typst.toml'
	if err != nil {
		typstIgnore = &ignore.GitIgnore{}
		logger.Warn("no '.typstignore' file. copy everything from", "cwd", cwd)
	}

	logger.Info("installing to", "target", dstDir)

	if isDryRun {
		logger.Warn("perform dry-run")
	}

	s := setupSpinner()
	s.Start()
	var payload []string
	err = filepath.WalkDir(cwd, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if typstIgnore.MatchesPath(path) {
			return nil
		}
		payload = append(payload, path)
		return nil
	})

	if err != nil {
		return err
	}

	if isDryRun {
		s.Stop()
		for _, src := range payload {
			dstFile := Must(paths.ResolveTargetPath(cwd, src, dstDir))
			logger.Debug("would copy", "src", src, "dst", dstFile)
		}
		return nil
	}

	maxSends := len(payload)
	errCh := make(chan error, maxSends)
	logCh := make(chan transferLog, maxSends)

	var wg sync.WaitGroup
	for _, src := range payload {
		wg.Go(func() {
			dstFile := Must(paths.ResolveTargetPath(cwd, src, dstDir))
			err := transferFile(src, dstFile, isEditable)
			logCh <- transferLog{src, dstFile}
			errCh <- err
		})
	}
	wg.Wait()
	close(errCh)
	close(logCh)
	s.Stop()

	for e := range errCh {
		if e != nil {
			logger.Error("an error occurred during the file transfer")
			return e
		}
	}

	for l := range logCh {
		logger.Debug("copy", "src", filepath.Base(l.src), "dst", l.dst)
	}

	logger.Infof("package '%s' successfully installed", pkg.Name)
	return nil
}

type transferLog struct {
	src string
	dst string
}

func transferFile(srcPath, dstPath string, isEditable bool) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		return err
	}
	var err error
	switch isEditable {
	case true:
		err = os.Symlink(srcPath, dstPath)
	case false:
		err = files.CopyFile(srcPath, dstPath)
	}
	if err != nil {
		return err
	}
	return nil
}

func getCurrentWorkingDir(args []string) string {
	if len(args) > 0 {
		cwd := Must(filepath.Abs(args[0]))
		return cwd
	}
	return Must(os.Getwd())
}
