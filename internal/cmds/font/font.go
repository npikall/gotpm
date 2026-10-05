// Package font implements the font command: it installs font families into the
// font directory, and declares and pins them for the current project.
package font

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/ui"
)

var ErrNotDeclared = errors.New("not a font of this project")

// Options holds the resolved font flags.
type Options struct {
	// Force replaces or deletes a family gotpm did not install.
	Force bool
}

// Install fetches the newest commit of a family into the font directory,
// without touching any project.
func Install(ctx context.Context, name string, opts *Options, logger *log.Logger) error {
	dir, err := fonts.OpenDir()
	if err != nil {
		return err
	}
	result, err := fetch(ctx, dir, name, opts, logger)
	if err != nil {
		return err
	}
	ui.Infof("installed font %q to %q", result.Pin.Family, dir.Path(result.Pin.Family))
	ui.Notes(result.Notes(dir))
	return nil
}

// Uninstall deletes a family from the font directory.
func Uninstall(name string, opts *Options, logger *log.Logger) error {
	dir, err := fonts.OpenDir()
	if err != nil {
		return err
	}
	logger.Debug("uninstalling font", "family", name, "path", dir.Path(name))
	if err := dir.Uninstall(name, opts.Force); err != nil {
		return err
	}
	ui.Infof("uninstalled font %q", name)
	return nil
}

// Search prints the families the font source offers that match query.
func Search(ctx context.Context, query string, refresh bool, logger *log.Logger) error {
	index, err := fonts.OpenIndex()
	if err != nil {
		return err
	}
	logger.Debug("searching fonts", "query", query, "cache", index.Path)
	found, err := ui.WithSpinner(" searching fonts", func() ([]fonts.Listing, error) {
		return index.Search(ctx, query, refresh)
	})
	if err != nil {
		return err
	}
	printListings(query, found)
	return nil
}

func printListings(query string, found []fonts.Listing) {
	if len(found) == 0 {
		ui.Warnf("no font matches %q", query)
	}
	for _, listing := range found {
		ui.Printf("%s (%s)\n", listing.Family, listing.License)
	}
}

// Add pins the newest commit of a family in the current project, installs it
// and declares it. The lock is written before the manifest, so an interrupted
// add leaves an undeclared pin, which the next sync prunes.
func Add(ctx context.Context, name string, opts *Options, logger *log.Logger) error {
	project, dir, err := open()
	if err != nil {
		return err
	}
	logger.Debug("adding font to project", "dir", project.Dir, "family", name)
	family := declaredAs(project.Fonts(), name)
	result, err := fetch(ctx, dir, family, opts, logger)
	if err != nil {
		return err
	}
	if err := record(project, family, result.Pin); err != nil {
		return err
	}
	ui.Infof("added font %q at %s", family, result.Pin.Hash)
	ui.Notes(result.Notes(dir))
	return nil
}

// Remove drops a family from the project's manifest and lock. Its files stay
// in the font directory, which other projects share.
func Remove(name string, logger *log.Logger) error {
	project, err := deps.OpenProject()
	if err != nil {
		return err
	}
	family, err := undeclare(project, name)
	if err != nil {
		return err
	}
	logger.Debug("removed font from the manifest", "dir", project.Dir, "family", family)
	stillPinned, err := pruneFonts(project, project.Fonts(), family)
	if err != nil {
		return err
	}
	reportRemoved(family, stillPinned)
	return nil
}

// undeclare drops a family from the manifest and returns the spelling it was
// declared under.
func undeclare(project *deps.Project, name string) (string, error) {
	declared := project.Fonts()
	i := slices.IndexFunc(declared, func(f string) bool { return fonts.Key(f) == fonts.Key(name) })
	if i < 0 {
		return "", fmt.Errorf("%w: %q is not listed in %s", ErrNotDeclared, name, manifest.FileName)
	}
	return declared[i], project.SetFonts(slices.Delete(slices.Clone(declared), i, i+1))
}

func reportRemoved(family string, stillPinned bool) {
	ui.Infof("removed font %q", family)
	if stillPinned {
		ui.Warnf("%q stays pinned: a dependency still requires it", family)
	}
	ui.Infof("the font files are still in the font directory; run 'gotpm font uninstall %q' to delete them", family)
}

func open() (*deps.Project, fonts.Dir, error) {
	project, err := deps.OpenProject()
	if err != nil {
		return nil, fonts.Dir{}, err
	}
	dir, err := fonts.OpenDir()
	return project, dir, err
}

// record pins font in the lock, then declares it in the manifest.
func record(project *deps.Project, family string, font lockfile.Font) error {
	if err := pin(project, font); err != nil {
		return err
	}
	return declare(project, family)
}

// fetch resolves the newest commit of a family and installs it.
func fetch(ctx context.Context, dir fonts.Dir, name string, opts *Options, logger *log.Logger) (fonts.Result, error) {
	src := fonts.Default()
	return ui.WithSpinner(fmt.Sprintf(" fetching font %q", name), func() (fonts.Result, error) {
		pin, err := src.Resolve(ctx, name)
		if err != nil {
			return fonts.Result{}, err
		}
		logger.Debug("resolved font", "family", name, "url", pin.URL, "hash", pin.Hash)
		pin.Direct = true
		return dir.Ensure(ctx, src, pin, opts.Force)
	})
}

// declaredAs is the spelling a family is declared under already, or name when
// the project does not declare it yet.
func declaredAs(declared []string, name string) string {
	for _, family := range declared {
		if fonts.Key(family) == fonts.Key(name) {
			return family
		}
	}
	return name
}

func pin(project *deps.Project, font lockfile.Font) error {
	lock, err := project.Lock()
	if err != nil {
		return err
	}
	lock.UpsertFont(font)
	return project.SaveLock(lock)
}

func declare(project *deps.Project, family string) error {
	declared := project.Fonts()
	if slices.Contains(declared, family) {
		return nil
	}
	return project.SetFonts(append(slices.Clone(declared), family))
}

// pruneFonts drops the pins nothing reaches any more, and reports whether
// family is still pinned because a dependency requires it.
func pruneFonts(project *deps.Project, declared []string, family string) (bool, error) {
	lock, err := project.Lock()
	if err != nil {
		return false, err
	}
	lock.PruneFonts(declared)
	if err := project.SaveLock(lock); err != nil {
		return false, err
	}
	_, pinned := lock.GetFont(family)
	return pinned, nil
}
