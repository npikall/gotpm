// Package typstsrc reads typst source files: which packages and which files
// they pull in, and rewriting the versions of the packages they import.
package typstsrc

import (
	"bufio"
	"fmt"
	"io"
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
	return scanLines(file, filepath.Dir(path), imports)
}

func scanLines(r io.Reader, baseDir string, imports *[]Import) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := scanLine(scanner.Text(), baseDir, imports); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("could not scan file: %w", err)
	}
	return nil
}

// scanLine records the import on line, or scans the file it includes.
func scanLine(line, baseDir string, imports *[]Import) error {
	if statement, ok := quoted(line, importPrefix); ok {
		*imports = append(*imports, newImport(statement, baseDir))
		return nil
	}
	if included, ok := quoted(line, includePrefix); ok {
		return scanInto(filepath.Join(baseDir, included), imports)
	}
	return nil
}

// quoted returns the string a line opens with prefix, when it is closed.
func quoted(line, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(line, prefix)
	if !ok {
		return "", false
	}
	value, _, ok := strings.Cut(rest, `"`)
	return value, ok
}

func newImport(statement, baseDir string) Import {
	imp := Import{Statement: statement}
	if !imp.IsPackage() {
		imp.File = filepath.Join(baseDir, statement)
	}
	return imp
}
