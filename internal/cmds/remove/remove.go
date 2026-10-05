// Package remove implements the remove command: it drops a dependency from the
// current project and, with it, the transitive dependencies nothing else needs
// any more.
package remove

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

var ErrNotDeclared = errors.New("not a dependency of this project")

// Options holds the resolved remove flags.
type Options struct {
	// Prune deletes the removed packages from the package directory as well. It
	// is opt-in because that directory is shared: another project may import
	// the same version.
	Prune bool
}

// Run removes the dependency written as imp, e.g. "@gotpm/cetz:0.3.1". The
// manifest is written before the lock, so an interrupted remove leaves a lock
// holding more than the project declares, which the next sync prunes.
func Run(imp string, opts *Options, logger *log.Logger) error {
	ref, err := parse(imp)
	if err != nil {
		return err
	}
	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	logger.Debug("removing from project", "dir", project.Dir, "package", ref)

	removed, err := drop(project, ref.String())
	if err != nil {
		return err
	}
	return finish(ref.String(), removed, opts, logger)
}

// drop removes imp from the manifest, then prunes the lock of everything that
// is no longer required, and returns what it pruned.
func drop(project *deps.Project, imp string) ([]lockfile.Entry, error) {
	remaining, err := withoutDependency(project, imp)
	if err != nil {
		return nil, err
	}
	if err := project.SetDependencies(remaining); err != nil {
		return nil, err
	}
	return pruneLock(project, remaining)
}

// finish deletes the removed packages' files when asked to, and reports.
func finish(imp string, removed []lockfile.Entry, opts *Options, logger *log.Logger) error {
	if opts.Prune {
		if err := uninstall(removed, logger); err != nil {
			return err
		}
	}
	report(imp, removed, opts.Prune)
	return nil
}

func parse(imp string) (pkg.Ref, error) {
	refs, err := manifest.ParseDependencies([]string{imp})
	if err != nil {
		return pkg.Ref{}, err
	}
	return refs[0], nil
}

func withoutDependency(project *deps.Project, imp string) ([]string, error) {
	declared := project.Dependencies()
	if !slices.Contains(declared, imp) {
		return nil, fmt.Errorf("%w: %s is not listed in %s", ErrNotDeclared, imp, manifest.FileName)
	}
	return slices.DeleteFunc(slices.Clone(declared), func(dep string) bool { return dep == imp }), nil
}

func pruneLock(project *deps.Project, remaining []string) ([]lockfile.Entry, error) {
	lock, err := project.Lock()
	if err != nil {
		return nil, err
	}
	removed := lock.Prune(remaining)
	lock.PruneFonts(project.Fonts())
	if err := project.SaveLock(lock); err != nil {
		return nil, err
	}
	return removed, nil
}

func uninstall(removed []lockfile.Entry, logger *log.Logger) error {
	s, err := store.OpenPackageDir()
	if err != nil {
		return err
	}
	for _, entry := range removed {
		if err := removeEntry(s, entry, logger); err != nil {
			return err
		}
	}
	return nil
}

func removeEntry(s store.Store, entry lockfile.Entry, logger *log.Logger) error {
	ref, err := pkg.New(entry.Namespace, entry.Name, entry.Version)
	if err != nil {
		return err
	}
	if err := s.Remove(ref); err != nil {
		return err
	}
	logger.Debug("deleted from the package directory", "package", ref, "path", s.Dir(ref))
	return nil
}

func report(imp string, removed []lockfile.Entry, pruned bool) {
	dropped, orphans := partition(imp, removed)

	ui.Infof("removed %s", ui.Package(imp))
	if !dropped {
		ui.Warnf("%s stays installed: another dependency still requires it", ui.Package(imp))
	}
	if len(orphans) > 0 {
		ui.Infof("  no longer needed: %s", strings.Join(orphans, ", "))
	}
	if hint := pruneHint(removed, pruned); hint != "" {
		ui.Infof("%s", hint)
	}
}

// partition reports whether imp itself was pruned from the lock, and the
// sorted imports of the others that went with it.
func partition(imp string, removed []lockfile.Entry) (bool, []string) {
	orphans := make([]string, 0, len(removed))
	dropped := false
	for _, entry := range removed {
		if entry.Import == imp {
			dropped = true
			continue
		}
		orphans = append(orphans, entry.Import)
	}
	slices.Sort(orphans)
	return dropped, orphans
}

func pruneHint(removed []lockfile.Entry, pruned bool) string {
	if len(removed) == 0 || pruned {
		return ""
	}
	return "the package files are still in the package directory; pass --prune to delete them"
}
