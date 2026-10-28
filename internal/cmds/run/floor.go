package run

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

// projectFloor is the project's floor: the highest of its own compiler requirement
// and those of its declared dependencies, read from their manifests in the
// package directory. A dependency's compiler pin means nothing to the project
// and is never read. Run never Syncs, so a declared dependency that is not
// installed contributes nothing; that is noted on out.
func projectFloor(project *deps.Project, out io.Writer, logger *log.Logger) (compilers.Requirement, error) {
	requirements := []compilers.Requirement{{Version: project.CompilerRequirement()}}
	declared := project.Dependencies()
	if len(declared) == 0 {
		return compilers.Floor(requirements)
	}
	packages, err := store.OpenPackageDir()
	if err != nil {
		return compilers.Requirement{}, err
	}
	for _, dep := range declared {
		// One at a time, so an invalid declared dependency, which Sync reports,
		// only goes uncounted instead of failing the compile.
		refs, err := manifest.ParseDependencies([]string{dep})
		if err != nil {
			logger.Debug("not counting an invalid dependency", "dependency", dep, "err", err)
			continue
		}
		ref := refs[0]
		m, err := manifest.LoadFile(filepath.Join(packages.Dir(ref), manifest.FileName))
		if errors.Is(err, fs.ErrNotExist) {
			ui.Fwarnf(out, "%s is not installed, so its compiler requirement is not counted; run 'gotpm sync' to install it",
				ui.Package(ref.String()))
			continue
		}
		if err != nil {
			return compilers.Requirement{}, fmt.Errorf("could not read the compiler requirement of %s: %w", ref, err)
		}
		requirements = append(requirements, compilers.Requirement{Version: m.Package.Compiler, Of: ref.String()})
	}
	return compilers.Floor(requirements)
}
