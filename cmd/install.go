/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
}

func installRunner(cmd *cobra.Command, args []string) error {
	sourceDir, err := resolveSourceDir(args)
	if err != nil {
		return err
	}
	fmt.Println(sourceDir)
	return nil
}

var ErrTooManyArguments = errors.New("too many arguments: expected one directory path")

func resolveSourceDir(args []string) (string, error) {
	numberOfArgs := len(args)
	if numberOfArgs > 1 {
		return "", ErrTooManyArguments
	}
	if numberOfArgs == 0 {
		return os.Getwd()
	}
	return resolveProvidedPath(args[0])
}

func resolveProvidedPath(rawPath string) (string, error) {
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", rawPath, err)
	}
	if err := validateIsDirectory(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func validateIsDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %q", path)
		}
		return fmt.Errorf("accessing path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %q", path)
	}
	return nil
}
