/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all locally installed Packages",
	Long:  "List all locally installed Packages",
	Example: `# list all available Packages
gotpm list`,
	RunE: listRunner,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
}

var ErrNoPackages = errors.New("no packages installed")

type pkgVersion struct {
	Name     string
	Editable bool
}

type installedPackage struct {
	Name     string
	Versions []pkgVersion
}

type packageNamespace struct {
	Name     string
	Packages []installedPackage
}

// isDirPath reports whether path is a directory, following symlinks.
func isDirPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// scanPackages walks root (namespace/package/version layout) and returns
// all installed packages, including editable (symlinked) versions.
func scanPackages(root string) ([]packageNamespace, error) {
	namespaceMap := make(map[string][]installedPackage)

	namespaceEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, nsEntry := range namespaceEntries {
		if !nsEntry.IsDir() {
			continue
		}

		namespaceName := nsEntry.Name()
		namespacePath := filepath.Join(root, namespaceName)

		packageEntries, err := os.ReadDir(namespacePath)
		if err != nil {
			continue
		}

		for _, pkgEntry := range packageEntries {
			if !pkgEntry.IsDir() {
				continue
			}

			packageName := pkgEntry.Name()
			packagePath := filepath.Join(namespacePath, packageName)

			versionEntries, err := os.ReadDir(packagePath)
			if err != nil {
				continue
			}

			var versions []pkgVersion
			for _, verEntry := range versionEntries {
				versionPath := filepath.Join(packagePath, verEntry.Name())
				if !isDirPath(versionPath) {
					continue
				}
				versions = append(versions, pkgVersion{
					Name:     verEntry.Name(),
					Editable: verEntry.Type()&fs.ModeSymlink != 0,
				})
			}

			if len(versions) == 0 {
				continue
			}

			sort.Slice(versions, func(i, j int) bool {
				return versions[i].Name < versions[j].Name
			})
			namespaceMap[namespaceName] = append(namespaceMap[namespaceName], installedPackage{
				Name:     packageName,
				Versions: versions,
			})
		}
	}

	var namespaces []packageNamespace
	for nsName, packages := range namespaceMap {
		sort.SliceStable(packages, func(i, j int) bool {
			return packages[i].Name < packages[j].Name
		})
		namespaces = append(namespaces, packageNamespace{
			Name:     nsName,
			Packages: packages,
		})
	}

	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})

	return namespaces, nil
}

func listRunner(cmd *cobra.Command, args []string) error {
	logger := setupLogger(cmd)

	typstPackagePath, err := resolveLocalPackageDirPath()
	if err != nil {
		return err
	}
	logger.Debug("looking in", "directory", typstPackagePath)

	if !isDirPath(typstPackagePath) {
		return ErrNoPackages
	}

	namespaces, err := scanPackages(typstPackagePath)
	if err != nil {
		return err
	}

	if len(namespaces) == 0 {
		logger.Info("no packages found")
		return nil
	}

	totalPackages := 0
	for _, ns := range namespaces {
		fmt.Println(StyleGreen.Render(fmt.Sprintf("@%s", ns.Name)))

		for _, pkg := range ns.Packages {
			totalPackages++
			printPackageWithVersions(pkg)
		}
	}

	footer := fmt.Sprintf("Total: %d packages across %d namespaces", totalPackages, len(namespaces))
	fmt.Println()
	fmt.Println(StyleMuted.Render(footer))
	return nil
}

func printPackageWithVersions(pkg installedPackage) {
	versions := pkg.Versions
	truncated := ""
	if len(versions) > 5 {
		truncated = fmt.Sprintf(" ... (+%d more)", len(versions)-5)
		versions = versions[:5]
	}

	var parts []string
	for _, v := range versions {
		if v.Editable {
			parts = append(parts, StyleYellow.Render(v.Name+" (editable)"))
		} else {
			parts = append(parts, StyleMuted.Render(v.Name))
		}
	}

	fmt.Printf("  %s %s%s\n",
		StyleNormal.Render(pkg.Name),
		strings.Join(parts, StyleMuted.Render(", ")),
		StyleMuted.Render(truncated),
	)
}
