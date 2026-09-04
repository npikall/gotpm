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

	remaining, err := withoutDependency(project, ref.String())
	if err != nil {
		return err
	}

	if err := project.SetDependencies(remaining); err != nil {
		return err
	}
	removed, err := pruneLock(project, remaining)
	if err != nil {
		return err
	}

	if opts.Prune {
		if err := uninstall(removed, logger); err != nil {
			return err
		}
	}
	report(ref.String(), removed, opts.Prune)
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
		ref, err := pkg.New(entry.Namespace, entry.Name, entry.Version)
		if err != nil {
			return err
		}
		if err := s.Remove(ref); err != nil {
			return err
		}
		logger.Debug("deleted from the package directory", "package", ref, "path", s.Dir(ref))
	}
	return nil
}

func report(imp string, removed []lockfile.Entry, pruned bool) {
	orphans := make([]string, 0, len(removed))
	dropped := false
	for _, entry := range removed {
		if entry.Import == imp {
			dropped = true
			continue
		}
		orphans = append(orphans, entry.Import)
	}

	ui.Infof("removed %s", ui.Package(imp))
	if !dropped {
		ui.Warnf("%s stays installed: another dependency still requires it", ui.Package(imp))
	}
	if len(orphans) > 0 {
		slices.Sort(orphans)
		ui.Infof("  no longer needed: %s", strings.Join(orphans, ", "))
	}
	if len(removed) > 0 && !pruned {
		ui.Infof("the package files are still in the package directory; pass --prune to delete them")
	}
}
