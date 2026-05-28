/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if all dependencies are available",
	Long:  `Check if all dependencies, that are imported by a file are available on the system`,
	RunE:  CheckRunner,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

var (
	ErrMissingFile = errors.New("expected file argument")
	ErrBadImport   = errors.New("not a valid import statement")
)

func CheckRunner(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return ErrMissingFile
	}
	importCh := make(chan string)
	done := make(chan struct{})

	go CheckImports(importCh, done)

	if err := ScanFileForImports(args[0], importCh); err != nil {
		return err
	}
	close(importCh)
	<-done
	// TODO: Print success message, if everything went good
	return nil
}

func CheckImports(linesCh <-chan string, done chan<- struct{}) {
	defer close(done)
	for line := range linesCh {
		if err := CheckImportStatement(line); err != nil {
			// TODO: use internal Print functionality to report bad imports
			fmt.Println("not found:", err)
		}
	}
}

func CheckImportStatement(imp string) error {
	switch {
	case strings.HasPrefix(imp, "@"):
		return CheckPackageExists(imp)
	default:
		return CheckFileExists(imp)
	}
}

func CheckPackageExists(imp string) error {
	namespace, identifier, found := strings.Cut(imp[len(`@`):], "\\")
	if !found {
		return ErrBadImport
	}
	pkg, version, found := strings.Cut(identifier, ":")
	if !found {
		return ErrBadImport
	}

	switch {
	case strings.HasPrefix(namespace, "@preview"):
		return CheckPackageInTypstUniverse(pkg, version)
	default:
		return CheckPackageExistsLocally(namespace, pkg, version)
	}
}

func CheckPackageExistsLocally(namespace, pkg, version string) error {
	// TODO: use internal functionality to check if package exists
	return nil
}

func CheckPackageInTypstUniverse(pkg, version string) error {
	// TODO: use internal functionality to check if package exists in typst index
	return nil
}

func CheckFileExists(path string) error {
	// TODO: use internal functionality to check if file exists
	return nil
}

func ScanFileForImports(path string, importCh chan<- string) error {
	file, err := os.Open(path) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer closeFile(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, `#import "`):
			if stmt, _, ok := strings.Cut(line[len(`#import "`):], `"`); ok {
				importCh <- stmt
			}
		case strings.HasPrefix(line, `#include "`):
			if nextFile, _, ok := strings.Cut(line[len(`#include "`):], `"`); ok {
				err := ScanFileForImports(nextFile, importCh)
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

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Fatal(err)
	}
}
