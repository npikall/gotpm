/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [file]",
	Short: "Check if all dependencies are available",
	Long:  `Check if all dependencies, that are imported by a file are available on the system`,
	RunE:  CheckRunner,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

var (
	ErrMissingFile       = errors.New("expected file argument")
	ErrBadImport         = errors.New("not a valid import statement")
	ErrPackageNotFound   = errors.New("package not found locally")
	ErrPackageNotInIndex = errors.New("package not found in typst universe")
	ErrFileNotFound      = errors.New("file not found")
)

// ResultItem is implemented by any diagnostic that can be collected in a CheckResult.
type ResultItem interface {
	Print()
	IsError() bool
}

// ImportIssue is a hard error — a dependency that could not be found.
type ImportIssue struct {
	Import string
	Err    error
}

func (i ImportIssue) Print()        { PrintMissingf("%q: %v", i.Import, i.Err) }
func (i ImportIssue) IsError() bool { return true }

// VersionWarning is a soft warning — import found but a newer version exists.
type VersionWarning struct {
	Package   string
	Requested string
	Latest    string
}

func (w VersionWarning) Print() {
	internal.PrintWarnf("package %q: requested %s, latest is %s", w.Package, w.Requested, w.Latest)
}
func (w VersionWarning) IsError() bool { return false }

type CheckResult struct {
	Items []ResultItem
}

func (r *CheckResult) Add(item ResultItem) {
	r.Items = append(r.Items, item)
}

func (r *CheckResult) HasIssues() bool {
	for _, item := range r.Items {
		if item.IsError() {
			return true
		}
	}
	return false
}

func (r *CheckResult) Print() {
	for _, item := range r.Items {
		item.Print()
	}
	if !r.HasIssues() {
		internal.PrintInfof("all imports ok")
	}
}

func CheckRunner(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return ErrMissingFile
	}
	importCh := make(chan string)
	resultCh := make(chan *CheckResult, 1)

	s := internal.SetupSpinner()
	s.Start()

	go CheckImports(importCh, resultCh)

	if err := ScanFileForImports(args[0], importCh); err != nil {
		close(importCh)
		<-resultCh
		s.Stop()
		return err
	}
	close(importCh)
	result := <-resultCh
	s.Stop()
	result.Print()
	return nil
}

func resolveIndex(ctx context.Context) (map[string]string, error) {
	cache, err := internal.LoadIndexCache()
	if err != nil || !cache.IsValid() {
		entries, fetchErr := internal.FetchTypstIndex(ctx)
		if fetchErr != nil {
			return nil, fmt.Errorf("could not fetch typst index: %w", fetchErr)
		}
		index := internal.BuildVersionIndex(entries)
		_ = internal.SaveIndexCache(index)
		return index, nil
	}
	return cache.Index, nil
}

func CheckImports(linesCh <-chan string, resultCh chan<- *CheckResult) {
	r := &CheckResult{}

	ctx, cancel := context.WithTimeout(context.Background(), internal.Timeout)
	defer cancel()
	index, indexErr := resolveIndex(ctx)

	seen := make(map[string]struct{})
	for line := range linesCh {
		if _, dup := seen[line]; dup {
			continue
		}
		seen[line] = struct{}{}
		if err := CheckImportStatement(line, r, index, indexErr); err != nil {
			r.Add(ImportIssue{Import: line, Err: err})
		}
	}
	resultCh <- r
}

func CheckImportStatement(imp string, result *CheckResult, index map[string]string, indexErr error) error {
	switch {
	case strings.HasPrefix(imp, "@"):
		return CheckPackageExists(imp, result, index, indexErr)
	default:
		return CheckFileExists(imp)
	}
}

func CheckPackageExists(imp string, result *CheckResult, index map[string]string, indexErr error) error {
	namespace, identifier, found := strings.Cut(imp[len(`@`):], "/")
	if !found {
		return ErrBadImport
	}
	pkg, version, found := strings.Cut(identifier, ":")
	if !found {
		return ErrBadImport
	}

	switch {
	case strings.HasPrefix(namespace, "preview"):
		return CheckPackageInTypstUniverse(pkg, version, result, index, indexErr)
	default:
		return CheckPackageExistsLocally(namespace, pkg, version)
	}
}

func CheckPackageExistsLocally(namespace, pkg, version string) error {
	dir, err := internal.ResolveLocalPackageDirPath()
	if err != nil {
		return fmt.Errorf("could not resolve local package directory: %w", err)
	}
	pkgPath := filepath.Join(dir, namespace, pkg, version)
	if _, err := os.Stat(pkgPath); err != nil {
		if os.IsNotExist(err) {
			return ErrPackageNotFound
		}
		return fmt.Errorf("could not stat package path %q: %w", pkgPath, err)
	}
	return nil
}

func CheckPackageInTypstUniverse(pkg, version string, result *CheckResult, index map[string]string, indexErr error) error {
	if indexErr != nil {
		return indexErr
	}
	latest, found := index[pkg]
	if !found {
		return ErrPackageNotInIndex
	}
	if latest != version {
		result.Add(VersionWarning{Package: pkg, Requested: version, Latest: latest})
	}
	return nil
}

func CheckFileExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %q", ErrFileNotFound, path)
		}
		return fmt.Errorf("could not stat file %q: %w", path, err)
	}
	return nil
}

func ScanFileForImports(path string, importCh chan<- string) error {
	file, err := os.Open(path) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer closeFile(file)

	baseDir := filepath.Dir(path)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, `#import "`):
			if stmt, _, ok := strings.Cut(line[len(`#import "`):], `"`); ok {
				if strings.HasPrefix(stmt, "@") {
					importCh <- stmt
				} else {
					importCh <- filepath.Join(baseDir, stmt)
				}
			}
		case strings.HasPrefix(line, `#include "`):
			if nextFile, _, ok := strings.Cut(line[len(`#include "`):], `"`); ok {
				err := ScanFileForImports(filepath.Join(baseDir, nextFile), importCh)
				if err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("could not scan file: %w", err)
	}
	return nil
}

func PrintMissingf(format string, a ...any) {
	prefix := internal.StyleRedBold.Render("missing")
	text := internal.StyleNormal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Fatal(err)
	}
}
