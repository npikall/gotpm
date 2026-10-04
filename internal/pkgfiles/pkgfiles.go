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

var ignoredNames = map[string]struct{}{
	".git":         {},
	".gitignore":   {},
	".typstignore": {},
	// A source directory copied as a package may itself be a former
	// gotpm-managed install (e.g. a fork checked out for `install <path>`),
	// carrying a provenance file that names a repository and commit the
	// files no longer match. Copying it forward would let the destination
	// inherit a record about content that isn't there.
	paths.ProvenanceFile: {},
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
	c := &collector{src: src, dst: dst, matcher: matcher}
	if err := filepath.WalkDir(src, c.visit); err != nil {
		return nil, fmt.Errorf("could not collect all files to transfer: %w", err)
	}
	return c.jobs, nil
}

// collector gathers a copy job for every file below src the matcher keeps.
type collector struct {
	src     string
	dst     string
	matcher *ignore.GitIgnore
	jobs    []Job
}

func (c *collector) visit(path string, d fs.DirEntry, walkErr error) error {
	rel, err := c.relative(path, walkErr)
	if err != nil {
		return err
	}
	if ShouldIgnore(rel, c.matcher) {
		return skip(d)
	}
	if !d.IsDir() {
		c.jobs = append(c.jobs, Job{Src: path, Dst: filepath.Join(c.dst, rel)})
	}
	return nil
}

// relative is path relative to src, unless walking to it failed.
func (c *collector) relative(path string, walkErr error) (string, error) {
	if walkErr != nil {
		return "", fmt.Errorf("walking %q: %w", path, walkErr)
	}
	rel, err := filepath.Rel(c.src, path)
	if err != nil {
		return "", fmt.Errorf("resolving relative path %q: %w", path, err)
	}
	return rel, nil
}

// skip leaves out an ignored entry, and everything below it for a directory.
func skip(d fs.DirEntry) error {
	if d.IsDir() {
		return filepath.SkipDir
	}
	return nil
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
	return writeCopy(srcFile, dest, info.Mode())
}

// writeCopy writes everything read from srcFile to dest, created with mode.
func writeCopy(srcFile *os.File, dest string, mode fs.FileMode) error {
	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) //nolint: gosec
	if err != nil {
		return fmt.Errorf("creating destination file %q: %w", dest, err)
	}
	defer destFile.Close() //nolint: errcheck

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copying %q to %q: %w", srcFile.Name(), dest, err)
	}
	return nil
}
