// Package run implements the run command: it stands in for the typst binary,
// choosing a compiler from the compiler directory, installing it when missing,
// and handing every argument to it unchanged.
package run

import (
	"context"
	"errors"
	"os"
	"sync"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/compilers"
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

// Run launches the compiler with args: the newest installed stable compiler,
// or the latest stable release, installed first, when none is installed.
func Run(ctx context.Context, args []string, logger *log.Logger) error {
	dir, err := compilers.OpenDir()
	if err != nil {
		return err
	}
	version, err := choose(ctx, dir, logger)
	if err != nil {
		return err
	}
	if err := ensure(ctx, dir, version, logger); err != nil {
		return err
	}
	binary, err := dir.Binary(version)
	if err != nil {
		return err
	}
	logger.Debug("launching compiler", "version", version, "binary", binary, "args", args)
	launcherMu.RLock()
	l := launcher
	launcherMu.RUnlock()
	return l(Launch{Binary: binary, Args: args, Env: os.Environ()})
}

// choose picks the newest installed stable compiler, asking the release source
// only when none is installed.
func choose(ctx context.Context, dir compilers.Dir, logger *log.Logger) (string, error) {
	installed, err := dir.Installed()
	if err != nil {
		return "", err
	}
	if newest := compilers.NewestStable(installed); newest != "" {
		logger.Debug("using the newest installed compiler", "version", newest)
		return newest, nil
	}
	latest, err := ui.WithSpinner(" looking up the latest compiler", func() (compilers.Release, error) {
		return compilers.Latest(ctx, compilers.Default())
	})
	if err != nil {
		return "", err
	}
	logger.Debug("no compiler installed, using the latest release", "version", latest.Version)
	return latest.Version, nil
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
