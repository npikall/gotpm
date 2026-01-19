/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/npikall/gotpm/internal/files"
	"github.com/npikall/gotpm/internal/list"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all locally installed Packages",
	Example: `gotpm list`,
	RunE:    listRunner,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
}

var ErrNoPackages = errors.New("no packages installed")

func listRunner(cmd *cobra.Command, args []string) error {
	logger := setupVerboseLogger(cmd)

	root, err := paths.GetTypstPackagePath()
	if err != nil {
		return err
	}
	logger.Debug("looking in", "directory", root)

	if !files.Exists(root) {
		return ErrNoPackages
	}

	namespaces, err := list.ScanPackages(root)
	if err != nil {
		return err
	}

	if len(namespaces) == 0 {
		logger.Info("no packages found")
		return nil
	}

	// Print packages
	totalPackages := 0

	for _, ns := range namespaces {
		fmt.Println(namespaceStyle.Render(fmt.Sprintf("@%s", ns.Name)))

		for _, pkg := range ns.Packages {
			totalPackages++

			versionStr := strings.Join(pkg.Versions, ", ")
			if len(pkg.Versions) > 5 {
				versionStr = strings.Join(pkg.Versions[:5], ", ") +
					fmt.Sprintf(" ... (+%d more)", len(pkg.Versions)-5)
			}

			fmt.Printf("  %s %s\n",
				packageStyle.Render(pkg.Name),
				versionStyle.Render(versionStr),
			)
		}
	}

	fmt.Println()
	fmt.Println(countStyle.Render(fmt.Sprintf("Total: %d packages across %d namespaces", totalPackages, len(namespaces))))
	return nil
}
