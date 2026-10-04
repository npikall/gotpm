// Package add implements the add command: it records a repository as a
// dependency of the current project, installs it together with everything it
// pulls in, and pins the lot in gotpm.lock.
package add

import (
	"errors"
	"fmt"
	"slices"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/ui"
)

// ErrUnresolvable is returned when a dependency declares a package add cannot
// find a repository for. Unlike install, add is strict on every reason: the lock
// it writes is public resolution data (ADR 0001).
var ErrUnresolvable = errors.New("cannot resolve dependency")

// Options holds the resolved add flags.
type Options struct {
	// Revision is the tag, branch or commit to pin. Empty asks for the newest
	// release.
	Revision string
	// Force replaces a package installed from another repository.
	Force bool
}

// Run adds the package at url to the current project. The lock is written
// before the manifest, so an interrupted add leaves an undeclared lock entry,
// which the next sync prunes, rather than a dependency nothing records.
func Run(url string, opts *Options, logger *log.Logger) error {
	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	logger.Debug("adding to project", "dir", project.Dir)

	entries, results, err := fetch(url, opts, logger)
	if err != nil {
		return err
	}
	if err := record(project, entries); err != nil {
		return err
	}
	report(results)
	return nil
}

// fetch resolves url and everything it depends on, and installs it all. The
// package at url comes first.
func fetch(url string, opts *Options, logger *log.Logger) ([]lockfile.Entry, []deps.Result, error) {
	walked, err := ui.WithSpinner("resolving "+url, func() (depgraph.Result, error) {
		return depgraph.Walk(resolve.Request{URL: url, Revision: opts.Revision}, depgraph.Options{}, logger)
	})
	if err != nil {
		return nil, nil, err
	}
	if err := errorForUnresolved(walked.Unresolved); err != nil {
		return nil, nil, err
	}
	results, err := installAll(walked.Entries, opts.Force, logger)
	return walked.Entries, results, err
}

func installAll(entries []lockfile.Entry, force bool, logger *log.Logger) ([]deps.Result, error) {
	installer, err := deps.OpenInstaller(force, logger)
	if err != nil {
		return nil, err
	}
	return ui.WithSpinner("installing", func() ([]deps.Result, error) {
		return installer.EnsureAll(entries)
	})
}

// record locks every entry and declares the first, the package that was added.
func record(project *deps.Project, entries []lockfile.Entry) error {
	if err := updateLock(project, entries); err != nil {
		return err
	}
	return declare(project, entries[0].Import)
}

func errorForUnresolved(unresolved []depgraph.Unresolved) error {
	if len(unresolved) == 0 {
		return nil
	}
	u := unresolved[0]
	reason := "it ships no " + lockfile.FileName
	if u.Reason == depgraph.IncompleteLock {
		reason = "its " + lockfile.FileName + " has no entry for it"
	}
	return fmt.Errorf("%w %s required by %s: %s"+
		"\nnote: %s must commit a %s recording where its dependencies come from",
		ErrUnresolvable, u.Dependency, u.RequiredBy, reason, u.RequiredBy, lockfile.FileName)
}

func updateLock(project *deps.Project, entries []lockfile.Entry) error {
	lock, err := project.Lock()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		lock.Upsert(entry)
	}
	return project.SaveLock(lock)
}

func declare(project *deps.Project, imp string) error {
	declared := project.Dependencies()
	if slices.Contains(declared, imp) {
		return nil
	}
	return project.SetDependencies(append(slices.Clone(declared), imp))
}

func report(results []deps.Result) {
	for i, result := range results {
		ui.Infof("%s", headline(i, result))
		ui.Notes(result.ReplacedNotice(), result.DriftWarning())
	}
}

// headline says what happened to the i-th result: the added package comes
// first, followed by what it pulled in.
func headline(i int, result deps.Result) string {
	ref := ui.Package(result.Ref.String())
	switch {
	case i > 0:
		return fmt.Sprintf("  %s (via %s)", ref, via(result.Entry))
	case result.Outcome == deps.UpToDate:
		return ref + " is already installed"
	default:
		return fmt.Sprintf("added %s from %s", ref, result.Entry.URL)
	}
}

func via(entry lockfile.Entry) string {
	if len(entry.RequiredBy) == 0 {
		return entry.URL
	}
	return entry.RequiredBy[0]
}
