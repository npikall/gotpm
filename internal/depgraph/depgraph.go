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

// maxDepth bounds the recursion. A chain this long is a mistake rather than a
// real dependency graph, and stopping with the chain named beats recursing
// until the process runs out of stack.
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

// Unresolved is one declared dependency Walk could not find a repository
// for. The subtree under it is skipped; everything else reachable is still
// walked. Walk itself takes no view on whether that makes the walk a
// failure — that is for the caller to decide from Reason.
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
	// RootNamespace overrides the namespace the root package is recorded
	// under. Transitive dependencies always resolve under manifest.Namespace
	// — a dependency's own source imports it as "@gotpm/...", so nothing
	// else is a namespace it could resolve under. Empty keeps
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
	w := &walker{logger: logger, opts: opts, index: make(map[string]int)}
	if err := w.visit(node{request: root, direct: true}, nil); err != nil {
		return Result{}, err
	}
	return Result{Entries: w.entries, Unresolved: w.unresolved}, nil
}

// node is one package to visit: where to get it, and what pulled it in.
type node struct {
	request resolve.Request
	// revision is the tag to record, when the pin came from a lock that knows
	// both a tag and the commit it pointed at. The request itself asks for the
	// commit, so a moved tag cannot change what gets installed.
	revision string
	// declaredAs is the import the dependant declared, kept to report a
	// dependency whose lock points at a version the package no longer holds.
	declaredAs string
	requiredBy string
	direct     bool
}

type walker struct {
	logger     *log.Logger
	opts       Options
	entries    []lockfile.Entry
	unresolved []Unresolved
	// index maps a visited repository commit to its position in entries. The
	// key is the commit rather than the import, because the same coordinate
	// coming from two different repositories is a conflict to be reported, not
	// a package already seen.
	index map[string]int
}

// visit resolves one node, records it and descends into its dependencies.
// chain holds the keys of the nodes currently being visited, which is what
// makes a cycle recognisable.
func (w *walker) visit(n node, chain []string) error {
	if len(chain) >= maxDepth {
		return fmt.Errorf("%w: more than %d levels: %s",
			ErrTooDeep, maxDepth, strings.Join(w.importsOf(chain), " -> "))
	}

	resolved, err := resolve.Resolve(n.request, w.logger)
	if err != nil {
		return w.wrapResolveError(n, err)
	}
	key := resolved.Source.Canonical + "@" + resolved.Hash

	// Descending again would not end. A lock can normally only name commits
	// that already exist, which makes a cycle hard to create by accident, but
	// a hand-written one costs nothing to survive.
	if i, seen := w.index[key]; seen {
		if slices.Contains(chain, key) {
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
	w.index[key] = len(w.entries)
	w.entries = append(w.entries, entry)

	children, unresolved, err := dependenciesOf(resolved, entry.Import)
	if err != nil {
		return err
	}
	w.unresolved = append(w.unresolved, unresolved...)
	childChain := append(slices.Clip(chain), key)
	for _, child := range children {
		if err := w.visit(child, childChain); err != nil {
			return err
		}
	}
	return nil
}

// importsOf names the packages a chain of visit keys stands for, so an error
// about the shape of the graph can describe it the way the user wrote it.
func (w *walker) importsOf(chain []string) []string {
	imports := make([]string, 0, len(chain))
	for _, key := range chain {
		if i, ok := w.index[key]; ok {
			imports = append(imports, w.entries[i].Import)
		}
	}
	return imports
}

// record folds a repeat visit into the entry already made for it: another
// dependant, and possibly the fact that it is declared directly as well.
func (w *walker) record(i int, n node) {
	entry := &w.entries[i]
	entry.Direct = entry.Direct || n.direct
	if n.requiredBy != "" && !slices.Contains(entry.RequiredBy, n.requiredBy) {
		entry.RequiredBy = append(entry.RequiredBy, n.requiredBy)
		slices.Sort(entry.RequiredBy)
	}
}

// newEntry turns a resolved repository into the lock entry that pins it. The
// root may be recorded under opts.RootNamespace; every other node is a
// transitive dependency and is always imported as "@gotpm/...", so nothing
// else is a namespace it could resolve under.
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

// dependenciesOf reads what a resolved package depends on, and looks each one
// up in the lock the package committed to learn where it comes from. A
// declared dependency missing from that lock comes back as an Unresolved
// rather than an error: whether that is fatal is a decision for Walk's
// caller, not the walker.
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

// pin is the revision a locked entry is fetched at: the exact commit, falling
// back to the tag for a lock written by hand.
func pin(entry lockfile.Entry) string {
	if entry.Hash != "" {
		return entry.Hash
	}
	return entry.Revision
}

// newUnresolved records the one failure users are likely to hit: a package
// that declares gotpm dependencies without publishing the lock that says
// where they live.
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

// wrapResolveError names the package a failed dependency was pulled in by,
// which is the only thing telling the user why gotpm went near that
// repository at all.
func (w *walker) wrapResolveError(n node, err error) error {
	if n.requiredBy == "" {
		return fmt.Errorf("resolving %s: %w", n.request.URL, err)
	}
	return fmt.Errorf("resolving %s required by %s: %w", n.declaredAs, n.requiredBy, err)
}

// warnOnCoordinateMismatch reports a dependant whose lock pins a commit that
// no longer holds the version it declares. The commit wins, because that is
// what the lock guarantees, but the dependant's import statements will not
// find the package under the name they use.
func (w *walker) warnOnCoordinateMismatch(entry lockfile.Entry, n node) {
	if n.declaredAs == "" || n.declaredAs == entry.Import {
		return
	}
	w.logger.Warn("locked commit holds a different version than the dependency declares",
		"required-by", n.requiredBy, "declared", n.declaredAs, "found", entry.Import,
		"url", entry.URL, "hash", entry.Hash)
}
