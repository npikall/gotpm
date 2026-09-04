// Package depgraph expands one repository argument into every package version
// it pulls in, directly or through its dependencies.
//
// Typst installs each version at its own path, so two versions of the same
// package coexist happily. That removes the hardest part of a dependency
// resolver: there is nothing to solve, only a graph to walk. What has to be
// discovered is where each transitive dependency lives, and the only place
// that is written down is the gotpm.lock a dependency committed alongside its
// own typst.toml — a dependency string such as "@gotpm/cetz:0.3.1" carries no
// repository.
package depgraph

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/resolve"
)

var ErrTooDeep = errors.New("dependency graph is too deep")

const maxDepth = 32

// Reason says why a declared dependency could not be resolved.
type Reason int

const (
	// NoLockShipped means the dependant's repository has no gotpm.lock at
	// all — nothing gotpm can do names where its dependencies live.
	NoLockShipped Reason = iota
	// IncompleteLock means the dependant ships a gotpm.lock, but it has no
	// entry for this declared dependency — a defect in the dependant.
	IncompleteLock
)

// Unresolved is one declared dependency Walk could not find a repository for.
// The subtree under it is skipped; whether that fails the walk is the caller's
// decision, taken from Reason.
type Unresolved struct {
	// Dependency is the unresolved import, e.g. "@gotpm/tidy:0.4.0".
	Dependency string
	// RequiredBy is the import of the package that declared it.
	RequiredBy string
	// RequiredByURL is RequiredBy's repository.
	RequiredByURL string
	Reason        Reason
}

// Options configures a Walk.
type Options struct {
	// RootNamespace overrides the namespace the root package is recorded under.
	// Transitive dependencies always resolve under manifest.Namespace. Empty keeps
	// manifest.Namespace for the root too.
	RootNamespace string
}

// Result is the outcome of a Walk.
type Result struct {
	// Entries holds one lock entry per resolved package version, the
	// requested one first. Direct and RequiredBy are filled in, so Entries
	// can be handed to (*lockfile.Lock).Upsert unchanged.
	Entries []lockfile.Entry
	// Unresolved holds one entry per declared dependency Walk could not find
	// a repository for.
	Unresolved []Unresolved
}

// Walk resolves a repository and everything it depends on.
func Walk(root resolve.Request, opts Options, logger *log.Logger) (Result, error) {
	w := &walker{logger: logger, opts: opts, bySourceCommit: make(map[string]int)}
	if err := w.visit(node{request: root, direct: true}, nil); err != nil {
		return Result{}, err
	}
	return Result{Entries: w.entries, Unresolved: w.unresolved}, nil
}

type node struct {
	request    resolve.Request
	revision   string
	declaredAs string
	requiredBy string
	direct     bool
}

type walker struct {
	logger         *log.Logger
	opts           Options
	entries        []lockfile.Entry
	unresolved     []Unresolved
	bySourceCommit map[string]int
}

func (w *walker) visit(n node, chain []string) error {
	if len(chain) >= maxDepth {
		return fmt.Errorf("%w: more than %d levels: %s",
			ErrTooDeep, maxDepth, strings.Join(w.importsOf(chain), " -> "))
	}

	resolved, err := resolve.Resolve(n.request, w.logger)
	if err != nil {
		return w.wrapResolveError(n, err)
	}
	sourceCommit := resolved.Source.Canonical + "@" + resolved.Hash

	if i, seen := w.bySourceCommit[sourceCommit]; seen {
		if slices.Contains(chain, sourceCommit) {
			w.logger.Debug("skipping dependency cycle", "package", w.entries[i].Import, "via", n.requiredBy)
		}
		w.record(i, n)
		return nil
	}

	entry, err := newEntry(resolved, n, w.opts)
	if err != nil {
		return err
	}
	w.warnOnCoordinateMismatch(entry, n)
	w.bySourceCommit[sourceCommit] = len(w.entries)
	w.entries = append(w.entries, entry)

	children, unresolved, err := dependenciesOf(resolved, entry.Import)
	if err != nil {
		return err
	}
	w.unresolved = append(w.unresolved, unresolved...)
	childChain := append(slices.Clip(chain), sourceCommit)
	for _, child := range children {
		if err := w.visit(child, childChain); err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) importsOf(chain []string) []string {
	imports := make([]string, 0, len(chain))
	for _, sourceCommit := range chain {
		if i, ok := w.bySourceCommit[sourceCommit]; ok {
			imports = append(imports, w.entries[i].Import)
		}
	}
	return imports
}

func (w *walker) record(i int, n node) {
	entry := &w.entries[i]
	entry.Direct = entry.Direct || n.direct
	if n.requiredBy != "" && !slices.Contains(entry.RequiredBy, n.requiredBy) {
		entry.RequiredBy = append(entry.RequiredBy, n.requiredBy)
		slices.Sort(entry.RequiredBy)
	}
}

func newEntry(resolved *resolve.Resolved, n node, opts Options) (lockfile.Entry, error) {
	namespace := manifest.Namespace
	if n.direct && opts.RootNamespace != "" {
		namespace = opts.RootNamespace
	}
	ref, err := resolved.Ref(namespace)
	if err != nil {
		return lockfile.Entry{}, fmt.Errorf("%s: %w", resolved.Source.Canonical, err)
	}

	revision := resolved.Revision
	if n.revision != "" {
		revision = n.revision
	}
	var requiredBy []string
	if n.requiredBy != "" {
		requiredBy = []string{n.requiredBy}
	}

	return lockfile.Entry{
		Import:     ref.String(),
		Name:       ref.Name,
		Version:    ref.Version.String(),
		Namespace:  ref.Namespace,
		URL:        resolved.Source.Canonical,
		Revision:   revision,
		Hash:       resolved.Hash,
		Direct:     n.direct,
		RequiredBy: requiredBy,
	}, nil
}

func dependenciesOf(resolved *resolve.Resolved, importing string) ([]node, []Unresolved, error) {
	declared := resolved.Manifest.Dependencies()
	if len(declared) == 0 {
		return nil, nil, nil
	}
	refs, err := manifest.ParseDependencies(declared)
	if err != nil {
		return nil, nil, fmt.Errorf("%s declares a dependency gotpm cannot install: %w", importing, err)
	}

	lock, err := lockfile.Load(resolved.Dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the lock of %s: %w", importing, err)
	}
	locked := paths.FileExists(lockfile.Path(resolved.Dir)) == nil

	var children []node
	var unresolved []Unresolved
	for _, ref := range refs {
		entry, ok := lock.Get(ref.String())
		if !ok {
			unresolved = append(unresolved, newUnresolved(importing, resolved.Source.Canonical, ref.String(), locked))
			continue
		}
		children = append(children, node{
			request:    resolve.Request{URL: entry.URL, Revision: pin(entry)},
			revision:   entry.Revision,
			declaredAs: ref.String(),
			requiredBy: importing,
		})
	}
	return children, unresolved, nil
}

func pin(entry lockfile.Entry) string {
	if entry.Hash != "" {
		return entry.Hash
	}
	return entry.Revision
}

func newUnresolved(importing, importingURL, dependency string, locked bool) Unresolved {
	reason := NoLockShipped
	if locked {
		reason = IncompleteLock
	}
	return Unresolved{
		Dependency:    dependency,
		RequiredBy:    importing,
		RequiredByURL: importingURL,
		Reason:        reason,
	}
}

func (w *walker) wrapResolveError(n node, err error) error {
	if n.requiredBy == "" {
		return fmt.Errorf("resolving %s: %w", n.request.URL, err)
	}
	return fmt.Errorf("resolving %s required by %s: %w", n.declaredAs, n.requiredBy, err)
}

func (w *walker) warnOnCoordinateMismatch(entry lockfile.Entry, n node) {
	if n.declaredAs == "" || n.declaredAs == entry.Import {
		return
	}
	w.logger.Warn("locked commit holds a different version than the dependency declares",
		"required-by", n.requiredBy, "declared", n.declaredAs, "found", entry.Import,
		"url", entry.URL, "hash", entry.Hash)
}
