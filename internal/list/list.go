package list

import (
	"os"
	"path/filepath"
	"sort"
)

type Package struct {
	Name     string
	Versions []string
}

type Namespace struct {
	Name     string
	Packages []Package
}

func ScanPackages(root string) ([]Namespace, error) {
	namespaceMap := make(map[string][]Package)

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

			var versions []string
			for _, verEntry := range versionEntries {
				if verEntry.IsDir() {
					versions = append(versions, verEntry.Name())
				}
			}
			if len(versions) > 0 {
				// Sort versions
				sort.Strings(versions)
				namespaceMap[namespaceName] = append(namespaceMap[namespaceName], Package{
					Name:     packageName,
					Versions: versions,
				})
			}
		}
	}
	var namespaces []Namespace
	for nsName, packages := range namespaceMap {
		sort.SliceStable(packages, func(i, j int) bool {
			return packages[i].Name < packages[j].Name
		})

		namespaces = append(namespaces, Namespace{
			Name:     nsName,
			Packages: packages,
		})
	}

	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})

	return namespaces, nil
}
