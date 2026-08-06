// Package pkgfiles decides which files make up a typst package and copies
// them. A package is everything below its root except what .gitignore or
// .typstignore excludes, plus a few names that are never part of a package.
package pkgfiles

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	ignore "github.com/sabhiram/go-gitignore"
)

// ignoredNames never belong to a package, whatever the ignore files say.
var ignoredNames = map[string]struct{}{
	".git":         {},
	".gitignore":   {},
	".typstignore": {},
}

// Job is a single file to transfer.
type Job struct {
	Src string
	Dst string
}

// CopyTree copies the package rooted at src into dst, honouring the ignore
// rules found in src.
func CopyTree(src, dst string) error {
	jobs, err := Collect(src, dst, Matcher(src))
	if err != nil {
		return err
	}
	return Run(jobs)
}

// Matcher builds the ignore rules of the package in dir from its .gitignore
// and .typstignore files. It returns nil when the package has neither.
func Matcher(dir string) *ignore.GitIgnore {
	typstIgnorePath := filepath.Join(dir, ".typstignore")
	extraLines := ReadIgnoreLines(typstIgnorePath)

	if m, err := manifest.LoadFrom(dir); err == nil {
		extraLines = append(extraLines, m.Package.Exclude...)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); err == nil {
		matcher, err := ignore.CompileIgnoreFileAndLines(gitIgnorePath, extraLines...)
		if err == nil {
			return matcher
		}
	}
	if len(extraLines) > 0 {
		return ignore.CompileIgnoreLines(extraLines...)
	}
	return nil
}

// ReadIgnoreLines reads the non-empty lines of an ignore file. A missing file
// yields no lines.
func ReadIgnoreLines(path string) []string {
	data, err := os.ReadFile(path) //nolint: gosec
	if err != nil {
		return nil
	}
	var lines []string
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// ShouldIgnore reports whether a path relative to the package root is excluded.
func ShouldIgnore(rel string, matcher *ignore.GitIgnore) bool {
	if rel == "." {
		return false
	}
	if _, ok := ignoredNames[filepath.Base(rel)]; ok {
		return true
	}
	if matcher != nil && matcher.MatchesPath(rel) {
		return true
	}
	return false
}

// Collect walks the package rooted at src and returns the files to transfer
// into dst. Ignored directories are skipped whole.
func Collect(src, dst string, matcher *ignore.GitIgnore) ([]Job, error) {
	var jobs []Job
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walking %q: %w", path, walkErr)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolving relative path %q: %w", path, err)
		}
		if ShouldIgnore(rel, matcher) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			jobs = append(jobs, Job{Src: path, Dst: filepath.Join(dst, rel)})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not collect all files to transfer: %w", err)
	}
	return jobs, nil
}

// Run performs every transfer job concurrently, reporting all failures.
func Run(jobs []Job) error {
	errCh := make(chan error, len(jobs))

	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Go(func() {
			if err := CopyFile(job.Src, job.Dst); err != nil {
				errCh <- err
			}
		})
	}
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CopyFile copies one file, creating parent directories and preserving the
// source file's mode.
func CopyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), paths.DirPerm); err != nil {
		return fmt.Errorf("creating parent directories for %q: %w", dest, err)
	}

	srcFile, err := os.Open(src) //nolint: gosec
	if err != nil {
		return fmt.Errorf("opening source file %q: %w", src, err)
	}
	defer srcFile.Close() //nolint: errcheck

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("reading file info %q: %w", src, err)
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()) //nolint: gosec
	if err != nil {
		return fmt.Errorf("creating destination file %q: %w", dest, err)
	}
	defer destFile.Close() //nolint: errcheck

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copying %q to %q: %w", src, dest, err)
	}
	return nil
}
