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

	"charm.land/lipgloss/v2"
	"github.com/npikall/gotpm/cmd/internal"
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

func sortVersionsByName(v []pkgVersion) {
	sort.Slice(v, func(i, j int) bool { return v[i].Name < v[j].Name })
}

func sortPackagesByName(p []installedPackage) {
	sort.SliceStable(p, func(i, j int) bool { return p[i].Name < p[j].Name })
}

func sortNamespacesByName(n []packageNamespace) {
	sort.Slice(n, func(i, j int) bool { return n[i].Name < n[j].Name })
}

func scanVersions(packagePath string) []pkgVersion {
	versionDirs, err := os.ReadDir(packagePath)
	if err != nil {
		return nil
	}

	var versions []pkgVersion
	for _, verDir := range versionDirs {
		versionPath := filepath.Join(packagePath, verDir.Name())
		if !isDirPath(versionPath) {
			continue
		}
		versions = append(versions, pkgVersion{
			Name:     verDir.Name(),
			Editable: verDir.Type()&fs.ModeSymlink != 0,
		})
	}
	return versions
}

func scanNamespacePackages(namespacePath string) []installedPackage {
	packageDirs, err := os.ReadDir(namespacePath)
	if err != nil {
		return nil
	}

	var packages []installedPackage
	for _, pkgDir := range packageDirs {
		if !pkgDir.IsDir() {
			continue
		}

		packagePath := filepath.Join(namespacePath, pkgDir.Name())
		versions := scanVersions(packagePath)
		if len(versions) == 0 {
			continue
		}

		sortVersionsByName(versions)
		packages = append(packages, installedPackage{
			Name:     pkgDir.Name(),
			Versions: versions,
		})
	}
	return packages
}

// scanPackages walks root (namespace/package/version layout) and returns
// all installed packages, including editable (symlinked) versions.
func scanPackages(root string) ([]packageNamespace, error) {
	namespaceDirs, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("could not read typst packages: %w", err)
	}

	packagesByNamespace := make(map[string][]installedPackage)
	for _, nsDir := range namespaceDirs {
		if !nsDir.IsDir() {
			continue
		}
		namespaceName := nsDir.Name()
		packages := scanNamespacePackages(filepath.Join(root, namespaceName))
		if len(packages) > 0 {
			packagesByNamespace[namespaceName] = packages
		}
	}

	var namespaces []packageNamespace
	for namespaceName, packages := range packagesByNamespace {
		sortPackagesByName(packages)
		namespaces = append(namespaces, packageNamespace{
			Name:     namespaceName,
			Packages: packages,
		})
	}

	sortNamespacesByName(namespaces)
	return namespaces, nil
}

func listRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)

	typstPackagePath, err := internal.ResolveLocalPackageDirPath()
	if err != nil {
		return fmt.Errorf("could not resolve package directory: %w", err)
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
		_, _ = lipgloss.Println(internal.StyleGreen.Render("@" + ns.Name))

		for _, pkg := range ns.Packages {
			totalPackages++
			printPackageWithVersions(pkg)
		}
	}

	footer := fmt.Sprintf("Total: %d packages across %d namespaces", totalPackages, len(namespaces))
	_, _ = lipgloss.Println()
	_, _ = lipgloss.Println(internal.StyleMuted.Render(footer))
	return nil
}

func printPackageWithVersions(pkg installedPackage) {
	versions := pkg.Versions
	truncated := ""
	maxDisplayedVersions := 5
	if len(versions) > maxDisplayedVersions {
		truncated = fmt.Sprintf(" ... (+%d more)", len(versions)-maxDisplayedVersions)
		versions = versions[:5]
	}

	var parts []string
	for _, v := range versions {
		if v.Editable {
			parts = append(parts, internal.StyleYellow.Render(v.Name+" (editable)"))
		} else {
			parts = append(parts, internal.StyleMuted.Render(v.Name))
		}
	}

	_, _ = lipgloss.Printf("  %s %s%s\n",
		internal.StyleNormal.Render(pkg.Name),
		strings.Join(parts, internal.StyleMuted.Render(", ")),
		internal.StyleMuted.Render(truncated),
	)
}
