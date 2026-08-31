// Package sync implements the sync command: it makes the package directory hold
// exactly what the current project's typst.toml and gotpm.lock describe.
//
// This is what turns a checkout into a working project. typst.toml is the
// authority on which packages the project wants; gotpm.lock is the authority on
// where each one lives and at which commit. sync reconciles the two against the
// store, and prunes lock entries nothing declares any more.
package sync

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/ui"
)

var (
	// ErrUnknownSource is a declared dependency the lock says nothing about.
	ErrUnknownSource = errors.New("declared dependency is missing from the lock")
	// ErrLockOutOfDate is a lock that --frozen forbids rewriting.
	ErrLockOutOfDate = errors.New("lock file is out of date")
)

// Options holds the resolved sync flags.
type Options struct {
	// Frozen refuses to rewrite the lock, so a stale one fails a CI run
	// instead of being quietly repaired.
	Frozen bool
	// Force overwrites a package installed from another repository.
	Force bool
}

// Run installs everything the current project depends on.
func Run(opts *Options, logger *log.Logger) error {
	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	logger.Debug("syncing project", "dir", project.Dir)

	lock, err := project.Lock()
	if err != nil {
		return err
	}
	declared, err := declaredImports(project)
	if err != nil {
		return err
	}
	if err := checkKnownSources(lock, declared); err != nil {
		return err
	}

	// Pruning also re-marks which entries are direct, so the lock can need
	// writing even when nothing was removed from it.
	wasDirect := lock.Direct()
	removed := lock.Prune(declared)
	changed := len(removed) > 0 || !slices.Equal(wasDirect, lock.Direct())

	if err := saveLock(project, lock, removed, changed, opts); err != nil {
		return err
	}

	installer, err := deps.OpenInstaller(opts.Force, logger)
	if err != nil {
		return err
	}
	results, err := ui.WithSpinner("installing", func() ([]deps.Result, error) {
		return installer.EnsureAll(lock.Packages)
	})
	if err != nil {
		return err
	}
	report(results, removed)
	return nil
}

// declaredImports reads the dependency list of the manifest, rejecting entries
// gotpm cannot install.
func declaredImports(project *deps.Project) ([]string, error) {
	refs, err := manifest.ParseDependencies(project.Dependencies())
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(refs))
	for _, ref := range refs {
		imports = append(imports, ref.String())
	}
	return imports, nil
}

// checkKnownSources rejects a dependency the project declares but the lock
// does not describe. Nothing can be done about it here: an import statement
// names a package, never the repository it comes from, so only the user knows
// where it should be fetched from.
func checkKnownSources(lock *lockfile.Lock, declared []string) error {
	var unknown []string
	for _, imp := range declared {
		if _, ok := lock.Get(imp); !ok {
			unknown = append(unknown, imp)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s"+
		"\nnote: %s records where a package comes from; run 'gotpm add <url>' to add it properly",
		ErrUnknownSource, strings.Join(unknown, ", "), lockfile.FileName)
}

// saveLock writes the lock back when pruning changed it, and refuses to when
// the caller asked for a frozen one.
func saveLock(project *deps.Project, lock *lockfile.Lock, removed []lockfile.Entry, changed bool, opts *Options) error {
	if !changed {
		return nil
	}
	if opts.Frozen {
		return fmt.Errorf("%w: %s no longer agrees with %s%s"+
			"\nnote: run 'gotpm sync' without --frozen and commit the result",
			ErrLockOutOfDate, lockfile.FileName, manifest.FileName, obsolete(removed))
	}
	return project.SaveLock(lock)
}

// obsolete names the entries nothing declares any more, when that is what the
// disagreement is.
func obsolete(removed []lockfile.Entry) string {
	if len(removed) == 0 {
		return ""
	}
	return " (no longer required: " + strings.Join(importsOf(removed), ", ") + ")"
}

// report prints what sync had to change, and says so plainly when it had to
// change nothing.
func report(results []deps.Result, removed []lockfile.Entry) {
	if len(removed) > 0 {
		ui.Infof("dropped from %s: %s", lockfile.FileName, strings.Join(importsOf(removed), ", "))
	}

	changed := 0
	for _, result := range results {
		if result.Outcome != deps.UpToDate {
			ui.Infof("installed %s", ui.Package(result.Ref.String()))
			changed++
		}
		if warning := result.DriftWarning(); warning != "" {
			ui.Warnf("%s", warning)
		}
	}
	if changed == 0 {
		ui.Infof("%d packages already up to date", len(results))
	}
}

func importsOf(entries []lockfile.Entry) []string {
	imports := make([]string, 0, len(entries))
	for _, entry := range entries {
		imports = append(imports, entry.Import)
	}
	slices.Sort(imports)
	return imports
}
