// Package lockfile reads and writes gotpm.lock, the file that pins every
// dependency of a project to an exact commit and records where it came from.
//
// The lock lives next to typst.toml and is meant to be committed: it is the
// only place that maps an import string like "@gotpm/cetz:0.3.1" back to the
// repository it was built from, which is what lets gotpm resolve the
// dependencies of a dependency.
package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/npikall/gotpm/internal/paths"
)

// SchemaVersion is the format version written into new lock files. Load
// refuses anything newer so an older gotpm does not silently misread a lock
// written by a newer one.
const SchemaVersion = 1

// FileName is the name of the lock file within a project directory.
const FileName = "gotpm.lock"

var (
	ErrInvalidLock   = errors.New("invalid 'gotpm.lock'")
	ErrUnknownFormat = errors.New("unsupported 'gotpm.lock' format version")
)

// Entry pins one package version to the exact commit it was installed from.
type Entry struct {
	// Import is the statement the package is imported with, e.g.
	// "@gotpm/cetz:0.3.1". It is the primary key of an entry.
	Import    string `json:"import"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	// URL is the repository the package was fetched from, without a scheme,
	// e.g. "github.com/johannes-wolf/cetz".
	URL string `json:"url"`
	// Revision is the tag or branch the user asked for. It is informational:
	// installs check out Hash so they stay reproducible when a tag moves.
	Revision string `json:"revision"`
	Hash     string `json:"hash"`
	// Subdir is the path of the package within the repository. Always empty
	// for now; packages must sit at the repository root.
	Subdir string `json:"subdir"`
	// Direct marks a dependency the project declares itself, as opposed to one
	// pulled in through another dependency.
	Direct bool `json:"direct"`
	// RequiredBy holds the imports of the packages that depend on this one. It
	// is empty for a dependency nothing else pulls in, and is what tells
	// Prune when a transitive dependency has become an orphan.
	RequiredBy []string `json:"required_by"`
}

// Lock is the whole content of a gotpm.lock file.
type Lock struct {
	Version  int     `json:"version"`
	Packages []Entry `json:"packages"`
}

// New returns an empty lock in the current format.
func New() *Lock {
	return &Lock{Version: SchemaVersion, Packages: []Entry{}}
}

// Path returns the location of the lock file for a project directory, without
// creating it.
func Path(projectDir string) string {
	return filepath.Join(projectDir, FileName)
}

// Load reads the lock file of a project. A project without a lock file yet is
// not an error: it loads as an empty lock.
func Load(projectDir string) (*Lock, error) {
	path := Path(projectDir)
	data, err := os.ReadFile(path) //nolint: gosec
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read %q: %w", path, err)
	}

	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("%w: %q: %w", ErrInvalidLock, path, err)
	}
	if lock.Version > SchemaVersion {
		return nil, fmt.Errorf("%w: %q is version %d, this gotpm understands up to %d",
			ErrUnknownFormat, path, lock.Version, SchemaVersion)
	}
	if lock.Packages == nil {
		lock.Packages = []Entry{}
	}
	return &lock, nil
}

// Save writes the lock to the project directory. Entries are sorted by import
// so the file has a stable order and produces readable diffs.
func Save(projectDir string, l *Lock) error {
	l.Version = SchemaVersion
	l.sort()

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal lock: %w", err)
	}
	return paths.WriteFile(Path(projectDir), append(data, '\n'))
}

// Get returns the entry for an import statement.
func (l *Lock) Get(imp string) (Entry, bool) {
	i := l.indexOf(imp)
	if i < 0 {
		return Entry{}, false
	}
	return l.Packages[i], true
}

// Direct returns the imports of the dependencies the project declares itself.
func (l *Lock) Direct() []string {
	var direct []string
	for _, entry := range l.Packages {
		if entry.Direct {
			direct = append(direct, entry.Import)
		}
	}
	slices.Sort(direct)
	return direct
}

// Upsert adds an entry, or updates the one already recorded for the same
// import. Provenance is taken from e, while Direct and RequiredBy accumulate:
// a package stays direct once it has been added directly, and keeps every
// dependant already known.
func (l *Lock) Upsert(e Entry) {
	i := l.indexOf(e.Import)
	if i < 0 {
		e.RequiredBy = normalise(e.RequiredBy)
		l.Packages = append(l.Packages, e)
		l.sort()
		return
	}

	existing := l.Packages[i]
	e.Direct = e.Direct || existing.Direct
	e.RequiredBy = normalise(append(existing.RequiredBy, e.RequiredBy...))
	l.Packages[i] = e
}

// Prune drops every entry that the given set of direct imports no longer
// reaches, and returns the entries it removed. An entry survives when it is
// itself declared direct, or when something that survives requires it, so
// removing one dependency also clears the transitive dependencies that only it
// pulled in.
//
// Prune also re-marks Direct from directImports, so a dependency demoted from
// direct to transitive is recorded correctly.
func (l *Lock) Prune(directImports []string) []Entry {
	direct := make(map[string]bool, len(directImports))
	for _, imp := range directImports {
		direct[imp] = true
	}
	for i := range l.Packages {
		l.Packages[i].Direct = direct[l.Packages[i].Import]
	}

	reachable := l.reachableFrom(direct)

	kept := make([]Entry, 0, len(l.Packages))
	var removed []Entry
	for _, entry := range l.Packages {
		if reachable[entry.Import] {
			kept = append(kept, entry)
			continue
		}
		removed = append(removed, entry)
	}

	// A surviving entry must not keep pointing at a dependant that is gone.
	for i := range kept {
		kept[i].RequiredBy = slices.DeleteFunc(kept[i].RequiredBy, func(imp string) bool {
			return !reachable[imp]
		})
	}

	l.Packages = kept
	slices.SortFunc(removed, byImport)
	return removed
}

// reachableFrom grows the set of roots by repeatedly adding every entry that
// something already in the set requires, until nothing new appears.
func (l *Lock) reachableFrom(roots map[string]bool) map[string]bool {
	reachable := make(map[string]bool, len(l.Packages))
	for _, entry := range l.Packages {
		if roots[entry.Import] {
			reachable[entry.Import] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, entry := range l.Packages {
			if reachable[entry.Import] {
				continue
			}
			for _, dependant := range entry.RequiredBy {
				if reachable[dependant] {
					reachable[entry.Import] = true
					changed = true
					break
				}
			}
		}
	}
	return reachable
}

func (l *Lock) indexOf(imp string) int {
	return slices.IndexFunc(l.Packages, func(e Entry) bool { return e.Import == imp })
}

func (l *Lock) sort() {
	slices.SortFunc(l.Packages, byImport)
}

func byImport(a, b Entry) int {
	switch {
	case a.Import < b.Import:
		return -1
	case a.Import > b.Import:
		return 1
	default:
		return 0
	}
}

// normalise sorts and deduplicates a dependant list so the lock stays stable
// no matter what order dependencies were resolved in.
func normalise(imports []string) []string {
	if len(imports) == 0 {
		return nil
	}
	sorted := slices.Clone(imports)
	slices.Sort(sorted)
	return slices.Compact(sorted)
}
