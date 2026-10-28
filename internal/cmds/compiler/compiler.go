// Package compiler implements the compiler command: it shows and deletes the
// compilers in the compiler directory.
package compiler

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/ui"
)

// List prints the installed compilers to out in version order, marking the
// newest stable one and, inside a project, its compiler pin, installed or not.
func List(out io.Writer, logger *log.Logger) error {
	dir, err := compilers.OpenDir()
	if err != nil {
		return err
	}
	logger.Debug("looking in", "directory", dir.Root)
	installed, err := dir.Installed()
	if err != nil {
		return err
	}
	pinned, err := projectPin()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		ui.Finfof(out, "no compiler installed")
	}
	newest := compilers.NewestStable(installed)
	for _, version := range withPin(installed, pinned) {
		_, _ = fmt.Fprintln(out, version+marks(version, newest, pinned, slices.Contains(installed, version)))
	}
	return nil
}

// Uninstall deletes a compiler from the compiler directory, pinned by some
// project or not.
func Uninstall(version string, out io.Writer, logger *log.Logger) error {
	dir, err := compilers.OpenDir()
	if err != nil {
		return err
	}
	logger.Debug("uninstalling compiler", "version", version, "path", dir.Path(version))
	if err := dir.Uninstall(version); err != nil {
		return err
	}
	ui.Finfof(out, "uninstalled compiler %s", version)
	return nil
}

// projectPin is the compiler pin of the project the working directory belongs
// to, "" when it pins none or there is no project.
func projectPin() (string, error) {
	project, err := deps.OpenProject()
	if errors.Is(err, manifest.ErrManifestNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return project.CompilerPin(), nil
}

// withPin is installed with pinned added in its place, when it is missing.
func withPin(installed []string, pinned string) []string {
	if pinned == "" || slices.Contains(installed, pinned) {
		return installed
	}
	versions := append(slices.Clone(installed), pinned)
	slices.SortFunc(versions, compilers.CompareVersions)
	return versions
}

// marks says what is special about version: the newest stable compiler, the
// project's compiler pin, and whether a pin is installed.
func marks(version, newest, pinned string, installed bool) string {
	var notes []string
	if version == newest {
		notes = append(notes, "newest")
	}
	if version == pinned {
		notes = append(notes, "pinned")
	}
	if !installed {
		notes = append(notes, "not installed")
	}
	if len(notes) == 0 {
		return ""
	}
	return " (" + strings.Join(notes, ", ") + ")"
}
