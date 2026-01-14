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
	"runtime"
	"strings"

	"github.com/npikall/gotpm/internal/list"
	"github.com/npikall/gotpm/internal/system"
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
}

var ErrNoPackages = errors.New("no packages installed")

func listRunner(cmd *cobra.Command, args []string) error {
	goos := runtime.GOOS
	homeDir := Must(os.UserHomeDir())
	root, err := system.GetTypstPath(goos, homeDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return ErrNoPackages
	}

	namespaces, err := list.ScanPackages(root)
	if err != nil {
		return err
	}

	if len(namespaces) == 0 {
		LogInfof("No packages found")
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
