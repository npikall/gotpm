// Package add implements the add command: it records a repository as a
// dependency of the current project, installs it together with everything it
// pulls in, and pins the lot in gotpm.lock.
package add

import (
	"slices"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

// Options holds the resolved add flags.
type Options struct {
	// Revision is the tag, branch or commit to pin. Empty asks for the newest
	// release.
	Revision string
	// Force replaces a package installed from another repository.
	Force bool
	// InstallDir overrides the package directory.
	InstallDir string
}

// Run adds the package at url to the current project.
func Run(url string, opts *Options, logger *log.Logger) error {
	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	logger.Debug("adding to project", "dir", project.Dir)

	entries, err := resolveGraph(url, opts.Revision, logger)
	if err != nil {
		return err
	}
	direct := entries[0]

	results, err := install(entries, opts, logger)
	if err != nil {
		return err
	}

	// The lock is written before the manifest, so an interrupted add leaves a
	// lock with an entry nothing declares — which the next sync prunes — rather
	// than a declared dependency whose repository nobody recorded, which no
	// command can recover from.
	if err := updateLock(project, entries); err != nil {
		return err
	}
	if err := declare(project, direct.Import); err != nil {
		return err
	}

	report(results)
	return nil
}

// resolveGraph expands the requested repository into every package version it
// pulls in.
func resolveGraph(url, revision string, logger *log.Logger) ([]lockfile.Entry, error) {
	spin := ui.Spinner("resolving " + url)
	spin.Start()
	entries, err := depgraph.Walk(resolve.Request{URL: url, Revision: revision}, logger)
	spin.Stop()
	return entries, err
}

// install puts every resolved package into the store.
func install(entries []lockfile.Entry, opts *Options, logger *log.Logger) ([]deps.Result, error) {
	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return nil, err
	}
	installer := deps.Installer{Store: s, Logger: logger, Force: opts.Force}

	spin := ui.Spinner("installing")
	defer spin.Stop()
	spin.Start()

	results := make([]deps.Result, 0, len(entries))
	for _, entry := range entries {
		result, err := installer.Ensure(entry)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// updateLock records every resolved package in the project's lock.
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

// declare adds the import to the manifest, unless it is already there.
//
// A package already declared at another version stays declared: typst installs
// every version at its own path, so both remain importable, and dropping the
// old line would silently break the source that imports it.
func declare(project *deps.Project, imp string) error {
	declared := project.Dependencies()
	if slices.Contains(declared, imp) {
		return nil
	}
	return project.SetDependencies(append(slices.Clone(declared), imp))
}

// report prints what was installed: the requested package, then everything it
// brought with it and which package asked for it.
func report(results []deps.Result) {
	for i, result := range results {
		switch {
		case i == 0 && result.Outcome == deps.UpToDate:
			ui.Infof("%s is already installed", ui.Package(result.Ref.String()))
		case i == 0:
			ui.Infof("added %s from %s", ui.Package(result.Ref.String()), result.Entry.URL)
		default:
			ui.Infof("  %s (via %s)", ui.Package(result.Ref.String()), via(result.Entry))
		}
		if warning := result.DriftWarning(); warning != "" {
			ui.Warnf("%s", warning)
		}
	}
}

// via names one package that asked for a transitive dependency.
func via(entry lockfile.Entry) string {
	if len(entry.RequiredBy) == 0 {
		return entry.URL
	}
	return entry.RequiredBy[0]
}
