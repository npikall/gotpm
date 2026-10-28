// Package run implements the run command: it stands in for the typst binary,
// choosing a compiler from the compiler directory, installing it when missing,
// and handing every argument to it unchanged.
package run

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/ui"
)

// Launch is what the launcher is asked to run.
type Launch struct {
	// Binary is the path of the compiler's binary.
	Binary string
	// Args are the arguments for the compiler, without the program name.
	Args []string
	// Env is the compiler's environment, as "key=value" pairs.
	Env []string
}

// Launcher hands control to a compiler. The real one does not return when it
// succeeds.
type Launcher func(Launch) error

var (
	launcherMu sync.RWMutex
	launcher   Launcher = launch
)

// UseLauncher replaces the launcher and returns a function restoring the
// previous one. It exists for tests.
func UseLauncher(l Launcher) func() {
	launcherMu.Lock()
	defer launcherMu.Unlock()
	previous := launcher
	launcher = l
	return func() {
		launcherMu.Lock()
		defer launcherMu.Unlock()
		launcher = previous
	}
}

// Run launches the compiler with args. Inside a project it is the project's
// compiler pin, chosen and written to the manifest when the project pins none
// or its pin is below the project's floor, the highest compiler requirement of
// the project and its installed declared dependencies; that is announced
// on out. Outside a project it is the newest installed stable compiler, or the
// latest stable release when none is installed. A missing compiler is
// installed first.
func Run(ctx context.Context, args []string, out io.Writer, logger *log.Logger) error {
	dir, err := compilers.OpenDir()
	if err != nil {
		return err
	}
	project, err := openProject()
	if err != nil {
		return err
	}
	required, pinned, err := floorAndPin(project, out, logger)
	if err != nil {
		return err
	}
	selection, err := choose(ctx, dir, required.Version, pinned, logger)
	if err != nil {
		return err
	}
	if err := ensure(ctx, dir, selection.Version, logger); err != nil {
		return err
	}
	if project != nil && selection.NewPin {
		if err := project.SetCompilerPin(selection.Version); err != nil {
			return err
		}
		announcePin(out, project.File, required, pinned, selection.Version)
	}
	binary, err := dir.Binary(selection.Version)
	if err != nil {
		return err
	}
	logger.Debug("launching compiler", "version", selection.Version, "binary", binary, "args", args)
	launcherMu.RLock()
	l := launcher
	launcherMu.RUnlock()
	return l(Launch{Binary: binary, Args: args, Env: os.Environ()})
}

// openProject finds the project the working directory belongs to, or nil
// outside any project.
func openProject() (*deps.Project, error) {
	project, err := deps.OpenProject()
	if errors.Is(err, manifest.ErrManifestNotFound) {
		return nil, nil //nolint: nilnil // no project is not an error for run
	}
	return project, err
}

// floorAndPin are the project's floor and compiler pin, neither outside a
// project.
func floorAndPin(project *deps.Project, out io.Writer, logger *log.Logger) (compilers.Requirement, string, error) {
	if project == nil {
		return compilers.Requirement{}, "", nil
	}
	required, err := projectFloor(project, out, logger)
	return required, project.CompilerPin(), err
}

// choose selects the compiler for a floor and pin, both "" outside a project,
// asking the release source only when no installed compiler will do.
func choose(ctx context.Context, dir compilers.Dir, floor, pinned string, logger *log.Logger) (compilers.Selection, error) {
	installed, err := dir.Installed()
	if err != nil {
		return compilers.Selection{}, err
	}
	releases := func() ([]compilers.Release, error) {
		return ui.WithSpinner(" looking up compiler releases", func() ([]compilers.Release, error) {
			return compilers.Stable(ctx, compilers.Default())
		})
	}
	selection, err := compilers.Select(floor, pinned, installed, releases)
	if err == nil {
		logger.Debug("chose compiler", "version", selection.Version, "floor", floor, "pin", pinned)
	}
	return selection, err
}

// announcePin says in one line that version was pinned in file, replacing
// previous, if any, for being below the floor.
func announcePin(out io.Writer, file string, floor compilers.Requirement, previous, version string) {
	if previous == "" {
		ui.Finfof(out, "pinned compiler %s in %s", version, file)
		return
	}
	ui.Finfof(out, "raised the compiler pin from %s to %s in %s: %s is below %s",
		previous, version, file, previous, floor)
}

// ensure installs version when it is missing, showing progress only then.
func ensure(ctx context.Context, dir compilers.Dir, version string, logger *log.Logger) error {
	if _, err := dir.Binary(version); !errors.Is(err, compilers.ErrNotInstalled) {
		return err
	}
	_, err := ui.WithSpinner(" installing compiler "+version, func() (bool, error) {
		return dir.Ensure(ctx, compilers.Default(), version)
	})
	if err == nil {
		logger.Debug("installed compiler", "version", version, "path", dir.Path(version))
	}
	return err
}
