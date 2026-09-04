// Package check implements the check command: it reports whether every
// dependency a typst file imports can actually be resolved.
package check

import (
	"context"
	"errors"
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/typstsrc"
	"github.com/npikall/gotpm/internal/ui"
)

var (
	ErrPackageNotFound   = errors.New("package not found locally")
	ErrPackageNotInIndex = errors.New("package not found in typst universe")
	ErrFileNotFound      = errors.New("file not found")
)

const previewNamespace = "preview"

// Issue is a dependency that could not be resolved.
type Issue struct {
	Import string
	Err    error
}

// Warning is a resolvable dependency that is not at its latest version.
type Warning struct {
	Package   string
	Requested string
	Latest    string
}

// Result collects what checking a file turned up.
type Result struct {
	Issues   []Issue
	Warnings []Warning
}

// Run checks every dependency imported by a file.
func Run(file string, log *log.Logger) error {
	result, err := ui.WithSpinner("", func() (Result, error) {
		return Analyze(file, log)
	})
	if err != nil {
		return err
	}

	render(result)
	return nil
}

// Analyze resolves every dependency of a file and reports what it found.
func Analyze(file string, log *log.Logger) (Result, error) {
	imports, err := typstsrc.ScanFile(file)
	if err != nil {
		return Result{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), index.Timeout)
	defer cancel()
	idx, indexErr := index.Load(ctx, index.Opts{})
	if indexErr != nil {
		log.Debug("could not load the package index", "err", indexErr)
	}

	root, err := paths.TypstPackagesDir()
	if err != nil {
		return Result{}, err
	}
	s := store.At(root)

	var result Result
	seen := make(map[string]struct{})
	for _, imp := range imports {
		if _, dup := seen[imp.Statement]; dup {
			continue
		}
		seen[imp.Statement] = struct{}{}

		if err := checkImport(imp, s, idx, indexErr, &result); err != nil {
			result.Issues = append(result.Issues, Issue{Import: imp.Statement, Err: err})
		}
	}
	return result, nil
}

func checkImport(imp typstsrc.Import, s store.Store, idx index.Index, indexErr error, result *Result) error {
	if !imp.IsPackage() {
		return checkFile(imp.File)
	}

	ref, err := pkg.ParseImport(imp.Statement)
	if err != nil {
		return err
	}
	if ref.Namespace == previewNamespace {
		return checkUniverse(ref, idx, indexErr, result)
	}
	if !s.Has(ref) {
		return ErrPackageNotFound
	}
	return nil
}

func checkUniverse(ref pkg.Ref, idx index.Index, indexErr error, result *Result) error {
	if indexErr != nil {
		return indexErr
	}
	latest, found := idx.Latest(ref.Name)
	if !found {
		return ErrPackageNotInIndex
	}
	if requested := ref.Version.String(); latest != requested {
		result.Warnings = append(result.Warnings, Warning{
			Package:   ref.Name,
			Requested: requested,
			Latest:    latest,
		})
	}
	return nil
}

func checkFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %q", ErrFileNotFound, path)
		}
		return fmt.Errorf("could not stat file %q: %w", path, err)
	}
	return nil
}

func render(result Result) {
	for _, issue := range result.Issues {
		ui.Missingf("%q: %v", issue.Import, issue.Err)
	}
	for _, warning := range result.Warnings {
		ui.Warnf("package %q: requested %s, latest is %s", warning.Package, warning.Requested, warning.Latest)
	}
	if len(result.Issues) == 0 {
		ui.Infof("all imports ok")
	}
}
