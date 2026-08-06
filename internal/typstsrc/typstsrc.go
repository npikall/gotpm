// Package typstsrc reads typst source files: which packages and which files
// they pull in, and rewriting the versions of the packages they import.
package typstsrc

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	importPrefix  = `#import "`
	includePrefix = `#include "`
)

// Import is one import statement found in a typst file.
type Import struct {
	// Statement is the imported thing exactly as written, e.g.
	// "@preview/cetz:0.5.2" or "utils/helper.typ".
	Statement string
	// File is the path a relative import resolves to, empty for packages.
	File string
}

// IsPackage reports whether the import refers to a package rather than a file.
func (i Import) IsPackage() bool {
	return strings.HasPrefix(i.Statement, "@")
}

// ScanFile lists the imports of a typst file, descending into every file it
// includes. Imports are returned in the order they are encountered.
func ScanFile(path string) ([]Import, error) {
	var imports []Import
	if err := scanInto(path, &imports); err != nil {
		return nil, err
	}
	return imports, nil
}

func scanInto(path string, imports *[]Import) error {
	file, err := os.Open(path) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close() //nolint: errcheck

	baseDir := filepath.Dir(path)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, importPrefix):
			statement, _, ok := strings.Cut(line[len(importPrefix):], `"`)
			if !ok {
				continue
			}
			*imports = append(*imports, newImport(statement, baseDir))
		case strings.HasPrefix(line, includePrefix):
			included, _, ok := strings.Cut(line[len(includePrefix):], `"`)
			if !ok {
				continue
			}
			if err := scanInto(filepath.Join(baseDir, included), imports); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("could not scan file: %w", err)
	}
	return nil
}

func newImport(statement, baseDir string) Import {
	imp := Import{Statement: statement}
	if !imp.IsPackage() {
		imp.File = filepath.Join(baseDir, statement)
	}
	return imp
}
