// Package add implements the add command: it records a repository as a
// dependency of the current project, installs it together with everything it
// pulls in, and pins the lot in gotpm.lock.
package add

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/remote"
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

	walked, results, err := fetch(url, opts, logger)
	if err != nil {
		return err
	}
	fontResults, err := record(project, walked, opts.Force, logger)
	if err != nil {
		return err
	}
	report(results)
	ui.Notes("", deps.UnreachableWarning(unreachablePins(walked.Unresolved), "the project"))
	fontResults.Report()
	return nil
}

// fetch resolves url and everything it depends on, and installs it all. The
// package at url comes first. Private repositories are not supported yet, so
// every clone is made without the user's credentials.
func fetch(url string, opts *Options, logger *log.Logger) (depgraph.Result, []deps.Result, error) {
	walked, err := ui.WithSpinner("resolving "+url, func() (depgraph.Result, error) {
		return depgraph.Walk(
			resolve.Request{URL: url, Revision: opts.Revision}, depgraph.Options{Anonymous: true}, logger,
		)
	})
	if errors.Is(err, remote.ErrRepositoryNotFound) {
		return depgraph.Result{}, nil, fmt.Errorf("%w"+
			"\nnote: if the repository is private: 'gotpm add' does not support private repositories yet;"+
			" 'gotpm install -r %s' installs it without adding it to the project", err, url)
	}
	if err != nil {
		return depgraph.Result{}, nil, err
	}
	if err := errorForUnresolved(walked.Unresolved); err != nil {
		return depgraph.Result{}, nil, err
	}
	results, err := installAll(walked.Entries, opts.Force, logger)
	return walked, results, err
}

func installAll(entries []lockfile.Entry, force bool, logger *log.Logger) ([]deps.Result, error) {
	installer, err := deps.OpenInstaller(force, logger)
	if err != nil {
		return nil, err
	}
	installer.Anonymous = true
	return ui.WithSpinner("installing", func() ([]deps.Result, error) {
		return installer.EnsureAll(entries)
	})
}

// record locks every entry and font, installs the fonts as they end up
// pinned, and declares the first entry, the package that was added.
func record(project *deps.Project, walked depgraph.Result, force bool, logger *log.Logger) (deps.FontResults, error) {
	fontResults, err := updateLock(project, walked, force, logger)
	if err != nil {
		return fontResults, err
	}
	return fontResults, declare(project, walked.Entries[0].Import)
}

// errorForUnresolved refuses the first dependency whose dependant does not
// say where it lives. An unreachable one is not refused: its pin is known.
func errorForUnresolved(unresolved []depgraph.Unresolved) error {
	i := slices.IndexFunc(unresolved, func(u depgraph.Unresolved) bool { return u.Reason != depgraph.Unreachable })
	if i < 0 {
		return nil
	}
	u := unresolved[i]
	reason := "it ships no " + lockfile.FileName
	if u.Reason == depgraph.IncompleteLock {
		reason = "its " + lockfile.FileName + " has no entry for it"
	}
	return fmt.Errorf("%w %s required by %s: %s"+
		"\nnote: %s must commit a %s recording where its dependencies come from",
		ErrUnresolvable, u.Dependency, u.RequiredBy, reason, u.RequiredBy, lockfile.FileName)
}

func updateLock(project *deps.Project, walked depgraph.Result, force bool, logger *log.Logger) (deps.FontResults, error) {
	lock, err := project.Lock()
	if err != nil {
		return deps.FontResults{}, err
	}
	lockAll(lock, walked)
	fontResults, err := deps.EnsureFonts(pinFonts(lock, walked.Fonts), force, logger)
	if err != nil {
		return fontResults, err
	}
	return fontResults, project.SaveLock(lock)
}

// lockAll pins every package the walk found, and the unreachable ones its
// dependants pin.
func lockAll(lock *lockfile.Lock, walked depgraph.Result) {
	for _, entry := range walked.Entries {
		lock.Upsert(entry)
	}
	for _, pin := range unreachablePins(walked.Unresolved) {
		lock.Upsert(pin)
	}
}

// pinFonts records the fonts the added packages declare and returns the pins
// that ended up in the lock. One the lock pins at another commit already keeps
// that commit, and the conflict is reported.
func pinFonts(lock *lockfile.Lock, pins []lockfile.Font) []lockfile.Font {
	kept := make([]lockfile.Font, 0, len(pins))
	for _, pin := range pins {
		recorded, conflict := lock.UpsertFont(pin)
		if conflict {
			ui.Warnf("font %q is pinned at %s, %s wants %s; keeping %s",
				pin.Family, deps.ShortHash(recorded.Hash), strings.Join(pin.RequiredBy, ", "),
				deps.ShortHash(pin.Hash), deps.ShortHash(recorded.Hash))
		}
		kept = append(kept, recorded)
	}
	return kept
}

// unreachablePins are the pins of the dependencies that could not be cloned.
// They are locked all the same: the pin is right, only this user cannot reach
// it.
func unreachablePins(unresolved []depgraph.Unresolved) []lockfile.Entry {
	var pins []lockfile.Entry
	for _, u := range unresolved {
		if u.Reason == depgraph.Unreachable {
			pins = append(pins, u.Pin)
		}
	}
	return pins
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
