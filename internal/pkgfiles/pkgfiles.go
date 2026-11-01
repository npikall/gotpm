// Package pkgfiles decides which files make up a typst package and copies
// them. A package's files are everything below its root except what its
// .gitignore files, its root .typstignore or its manifest's exclude leave out,
// and except a few names that are never part of a package.
package pkgfiles

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/format/gitignore"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"golang.org/x/sync/errgroup"
)

var ignoredNames = map[string]struct{}{
	".git":         {},
	".jj":          {},
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
	jobs, err := Collect(src, dst)
	if err != nil {
		return err
	}
	return Run(jobs)
}

// Collect walks the package rooted at src and returns the files to transfer
// into dst. Excluded directories are skipped whole, as git skips them: nothing
// below an excluded directory can be re-included.
func Collect(src, dst string) ([]Job, error) {
	rules, err := loadRules(src)
	if err != nil {
		return nil, err
	}
	c := &collector{src: src, dst: dst, rules: rules}
	if err := filepath.WalkDir(src, c.visit); err != nil {
		return nil, fmt.Errorf("could not collect all files to transfer: %w", err)
	}
	return c.jobs, nil
}

// collector gathers a copy job for every file below src the rules keep.
type collector struct {
	src   string
	dst   string
	rules *rules
	jobs  []Job
}

func (c *collector) visit(path string, d fs.DirEntry, walkErr error) error {
	rel, err := c.relative(path, walkErr)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil // loadRules already read the root's rules.
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if c.rules.excludes(parts, d.IsDir()) {
		return skip(d)
	}
	if !d.IsDir() {
		c.jobs = append(c.jobs, Job{Src: path, Dst: filepath.Join(c.dst, rel)})
		return nil
	}
	// The walk visits a directory before anything in it, so its .gitignore
	// joins the rules after its parents' and before its children's.
	return c.rules.addGitIgnore(path, parts)
}

// rules decide which paths below a package root are left out of the package.
// Only files in the package decide what is in it, so git's machine-local
// excludes never apply.
type rules struct {
	// git holds .gitignore patterns from the root down, in increasing
	// priority, so a deeper file outranks a shallower one.
	git []pattern
	// overrides holds the root .typstignore and the manifest's exclude. They
	// are the package's last word and outrank every .gitignore.
	overrides []pattern
}

// loadRules reads the ignore rules at the root of the package in dir: its
// .gitignore, its .typstignore and its manifest's exclude. A .gitignore
// further down joins as Collect walks into its directory.
func loadRules(dir string) (*rules, error) {
	r := &rules{}
	if err := r.addGitIgnore(dir, nil); err != nil {
		return nil, err
	}

	lines, err := readIgnoreFile(filepath.Join(dir, ".typstignore"))
	if err != nil {
		return nil, err
	}
	exclude, err := manifestExclude(dir)
	if err != nil {
		return nil, err
	}
	r.overrides = parsePatterns(append(lines, exclude...), nil)
	return r, nil
}

// excludes reports whether the path with the given components below the
// package root is left out of the package. isDir decides patterns ending in
// "/".
func (r *rules) excludes(path []string, isDir bool) bool {
	if _, ok := ignoredNames[path[len(path)-1]]; ok {
		return true
	}
	if m := lastMatch(r.overrides, path, isDir); m != gitignore.NoMatch {
		return m == gitignore.Exclude
	}
	return lastMatch(r.git, path, isDir) == gitignore.Exclude
}

// addGitIgnore adds the patterns of the .gitignore in dir, scoped to domain,
// the components of dir below the package root.
func (r *rules) addGitIgnore(dir string, domain []string) error {
	lines, err := readIgnoreFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return err
	}
	r.git = append(r.git, parsePatterns(lines, domain)...)
	return nil
}

// lastMatch is the result of the last pattern that matches path, so later
// patterns outrank earlier ones.
func lastMatch(patterns []pattern, path []string, isDir bool) gitignore.MatchResult {
	for _, p := range slices.Backward(patterns) {
		if m := p.match(path, isDir); m != gitignore.NoMatch {
			return m
		}
	}
	return gitignore.NoMatch
}

// pattern is one gitignore pattern, matched the way git matches it.
type pattern struct {
	gitignore.Pattern
	// belowOnly marks a pattern ending in "/**". Git matches it against what
	// is below a directory but never the directory itself; go-git also matches
	// the directory, which would prune it before a later negation such as
	// "!docs/keep.md" is consulted.
	belowOnly bool
}

func parsePatterns(lines, domain []string) []pattern {
	patterns := make([]pattern, 0, len(lines))
	for _, line := range lines {
		patterns = append(patterns, pattern{
			Pattern:   gitignore.ParsePattern(line, domain),
			belowOnly: strings.HasSuffix(strings.TrimRight(line, " "), "/**"),
		})
	}
	return patterns
}

func (p pattern) match(path []string, isDir bool) gitignore.MatchResult {
	if !p.belowOnly {
		return p.Match(path, isDir)
	}
	// A path is below the directory exactly when its parent is the directory
	// or below it, which is what go-git matches.
	parent := path[:len(path)-1]
	if len(parent) == 0 {
		return gitignore.NoMatch
	}
	return p.Match(parent, true)
}

// readIgnoreFile reads the patterns of an ignore file: its lines, less blank
// lines and comments. A missing file has none; any other failure to read it
// is an error, since skipping it would put files into the package that its
// author left out.
func readIgnoreFile(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint: gosec
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading ignore file %q: %w", path, err)
	}
	var lines []string
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// manifestExclude is the exclude list of the manifest at the root of the
// package in dir. A package without a manifest there excludes nothing; a
// manifest that cannot be read is an error, as an unreadable ignore file is.
func manifestExclude(dir string) ([]string, error) {
	m, err := manifest.LoadFile(filepath.Join(dir, manifest.FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading the package's exclude list: %w", err)
	}
	return m.Package.Exclude, nil
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

// Run performs the transfer jobs concurrently, at most one per CPU at a time,
// reporting all failures. Unbounded concurrency makes the filesystem the
// bottleneck: thousands of simultaneous opens and mkdirs contend in the kernel.
func Run(jobs []Job) error {
	errs := make([]error, len(jobs))

	var g errgroup.Group
	g.SetLimit(runtime.NumCPU())
	for i, job := range jobs {
		g.Go(func() error {
			errs[i] = CopyFile(job.Src, job.Dst)
			return nil
		})
	}
	_ = g.Wait() // every goroutine returns nil; failures are collected in errs

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
