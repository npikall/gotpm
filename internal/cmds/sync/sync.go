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

	lock, removed, err := reconcileLock(project, opts)
	if err != nil {
		return err
	}
	results, err := installAll(lock, opts.Force, logger)
	if err != nil {
		return err
	}
	report(results, removed)
	return nil
}

// reconcileLock brings the project's lock in line with what its manifest
// declares, and returns it with the entries it dropped.
func reconcileLock(project *deps.Project, opts *Options) (*lockfile.Lock, []lockfile.Entry, error) {
	lock, err := project.Lock()
	if err != nil {
		return nil, nil, err
	}
	declared, err := declaredImports(project)
	if err != nil {
		return nil, nil, err
	}
	if err := checkKnownSources(lock, declared); err != nil {
		return nil, nil, err
	}
	removed, changed := prune(lock, declared)
	return lock, removed, saveLock(project, lock, removed, changed, opts)
}

// prune drops what nothing declares any more, and reports whether the lock
// changed, which includes a package moving between direct and indirect.
func prune(lock *lockfile.Lock, declared []string) ([]lockfile.Entry, bool) {
	wasDirect := lock.Direct()
	removed := lock.Prune(declared)
	return removed, len(removed) > 0 || !slices.Equal(wasDirect, lock.Direct())
}

func installAll(lock *lockfile.Lock, force bool, logger *log.Logger) ([]deps.Result, error) {
	installer, err := deps.OpenInstaller(force, logger)
	if err != nil {
		return nil, err
	}
	return ui.WithSpinner("installing", func() ([]deps.Result, error) {
		return installer.EnsureAll(lock.Packages)
	})
}

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

func obsolete(removed []lockfile.Entry) string {
	if len(removed) == 0 {
		return ""
	}
	return " (no longer required: " + strings.Join(importsOf(removed), ", ") + ")"
}

func report(results []deps.Result, removed []lockfile.Entry) {
	reportDropped(removed)
	changed := 0
	for _, result := range results {
		if reportResult(result) {
			changed++
		}
	}
	if changed == 0 {
		ui.Infof("%d packages already up to date", len(results))
	}
}

func reportDropped(removed []lockfile.Entry) {
	if len(removed) > 0 {
		ui.Infof("dropped from %s: %s", lockfile.FileName, strings.Join(importsOf(removed), ", "))
	}
}

// reportResult tells what happened to one package, and whether it changed.
func reportResult(result deps.Result) bool {
	changed := result.Outcome != deps.UpToDate
	if changed {
		ui.Infof("installed %s", ui.Package(result.Ref.String()))
	}
	ui.Notes(result.ReplacedNotice(), result.DriftWarning())
	return changed
}

func importsOf(entries []lockfile.Entry) []string {
	imports := make([]string, 0, len(entries))
	for _, entry := range entries {
		imports = append(imports, entry.Import)
	}
	slices.Sort(imports)
	return imports
}
