// Package deps puts the packages a project's lock pins into the machine's
// package store.
//
// It is what add and sync have in common. Both end up holding lock entries and
// needing the store to match them; they differ only in where those entries came
// from. The store is shared by every project on the machine, so an install is
// never a plain copy: a directory that already holds a package version has to
// be shown to be the same package from the same repository before it is left
// alone or overwritten.
package deps

import (
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/git"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/store"
)

var ErrSourceConflict = errors.New("package installed from a different source")

// Outcome is what had to be done to make the store match a lock entry.
type Outcome int

const (
	// UpToDate means the pinned commit was already installed.
	UpToDate Outcome = iota
	// Installed means nothing was there and the package was fetched.
	Installed
	// Replaced means something else was there and was overwritten.
	Replaced
)

// Result reports what happened to one package.
type Result struct {
	Ref     pkg.Ref
	Entry   lockfile.Entry
	Outcome Outcome
	// MovedTag is the commit the entry's revision points at now, set only when
	// that is no longer the commit the lock pinned. The lock still wins; this
	// is what tells the user their pin has drifted from the tag it was made
	// from.
	MovedTag string
}

// DriftWarning describes a pin whose tag has moved since it was made, or the
// empty string when it has not. The locked commit is what was installed, so
// this is a report, not a failure.
func (r Result) DriftWarning() string {
	if r.MovedTag == "" {
		return ""
	}
	return fmt.Sprintf("%s of %s now points at %s, not the locked %s; installed the locked commit"+
		"\nnote: run 'gotpm add %s' to pin the current one",
		r.Entry.Revision, r.Entry.URL, ShortHash(r.MovedTag), ShortHash(r.Entry.Hash), r.Entry.URL)
}

// shortHashLen is how much of a commit hash is worth showing.
const shortHashLen = 8

// ShortHash abbreviates a commit hash for a message.
func ShortHash(hash string) string {
	if len(hash) <= shortHashLen {
		return hash
	}
	return hash[:shortHashLen]
}

// Installer makes the store hold what a set of lock entries pins.
type Installer struct {
	Store  store.Store
	Logger *log.Logger
	// Force overwrites a package installed from another repository instead of
	// refusing to touch it.
	Force bool
}

// OpenInstaller opens the package store and returns an installer writing into
// it. An empty installDir uses the store every project on the machine shares.
func OpenInstaller(installDir string, force bool, logger *log.Logger) (Installer, error) {
	s, err := store.Open(installDir)
	if err != nil {
		return Installer{}, err
	}
	return Installer{Store: s, Logger: logger, Force: force}, nil
}

// EnsureAll makes the store hold every package version a set of entries pins,
// in the order they are given.
//
// It stops at the first entry it cannot install and reports nothing about the
// ones before it: a lock describes a set of packages, and a half-installed set
// is not a state worth reporting on. Running the command again picks up where
// this one stopped, because Ensure leaves what is already installed alone.
func (i Installer) EnsureAll(entries []lockfile.Entry) ([]Result, error) {
	results := make([]Result, 0, len(entries))
	for _, entry := range entries {
		result, err := i.Ensure(entry)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Ensure makes the store hold exactly the package version an entry pins,
// fetching it when it is missing or is not the commit that was locked.
//
// A directory installed from another repository, or one gotpm did not install
// at all, is left alone and reported: other projects on this machine import it
// under the same coordinate.
func (i Installer) Ensure(entry lockfile.Entry) (Result, error) {
	ref, err := pkg.New(entry.Namespace, entry.Name, entry.Version)
	if err != nil {
		return Result{}, err
	}

	present, prov, err := i.inspect(ref, entry)
	if err != nil {
		return Result{}, err
	}
	if present == current {
		i.Logger.Debug("already installed", "package", ref, "hash", entry.Hash)
		return Result{Ref: ref, Entry: entry, Outcome: UpToDate}, nil
	}
	if present == foreign && !i.Force {
		return Result{}, conflictError(ref, entry, prov)
	}
	if present != absent {
		if err := i.Store.Remove(ref); err != nil {
			return Result{}, fmt.Errorf("could not remove %s before reinstalling it: %w", ref, err)
		}
	}

	moved, err := i.install(ref, entry)
	if err != nil {
		return Result{}, err
	}
	outcome := Installed
	if present != absent {
		outcome = Replaced
	}
	return Result{Ref: ref, Entry: entry, Outcome: outcome, MovedTag: moved}, nil
}

// state is what the store already holds for a package version.
type state int

const (
	// absent: nothing is installed under this coordinate.
	absent state = iota
	// current: the pinned commit, from the pinned repository.
	current
	// stale: the pinned repository, but another commit.
	stale
	// foreign: another repository, or nothing saying which.
	foreign
)

// inspect compares what is installed against what an entry pins.
func (i Installer) inspect(ref pkg.Ref, entry lockfile.Entry) (state, store.Provenance, error) {
	if !i.Store.Has(ref) {
		return absent, store.Provenance{}, nil
	}
	prov, ok, err := i.Store.ReadProvenance(ref)
	if err != nil {
		return absent, prov, err
	}
	switch {
	case !ok || prov.URL != entry.URL:
		return foreign, prov, nil
	case prov.Hash == entry.Hash:
		return current, prov, nil
	default:
		return stale, prov, nil
	}
}

// install fetches the pinned commit and copies it into the store.
func (i Installer) install(ref pkg.Ref, entry lockfile.Entry) (string, error) {
	src, err := resolve.Normalize(entry.URL)
	if err != nil {
		return "", err
	}
	clone, err := remote.EnsureClone(src.Canonical, src.CloneURL)
	if err != nil {
		return "", fmt.Errorf("could not fetch %s: %w", entry.URL, err)
	}
	defer clone.Repo.Close() //nolint: errcheck

	moved := movedTag(clone.Repo, entry)
	if err := clone.Repo.CheckoutRevision(entry.Hash); err != nil {
		return "", fmt.Errorf("could not check out %s of %s: %w", entry.Hash, entry.URL, err)
	}
	if err := i.Store.Install(ref, clone.Dir); err != nil {
		return "", err
	}
	i.Logger.Debug("installed", "package", ref, "url", entry.URL, "hash", entry.Hash)

	return moved, i.Store.WriteProvenance(ref, store.Provenance{
		URL: entry.URL, Revision: entry.Revision, Hash: entry.Hash,
	})
}

// movedTag reports the commit an entry's revision points at now, when that is
// no longer the commit the lock pinned.
//
// A revision that is a commit, or the moving HEAD of a repository without
// releases, cannot drift in a way worth reporting, and one that has since
// disappeared is not something the install needs to care about: the lock names
// a commit, and that is what gets installed either way.
func movedTag(repo *git.Repo, entry lockfile.Entry) string {
	if entry.Revision == "" || entry.Revision == "HEAD" || entry.Revision == entry.Hash {
		return ""
	}
	hash, err := repo.ResolveHash(entry.Revision)
	if err != nil || hash == entry.Hash {
		return ""
	}
	return hash
}

// conflictError describes a coordinate that is taken by a package from
// somewhere else. The store is machine-global, so overwriting it would change
// what another project imports; that has to be the user's decision.
func conflictError(ref pkg.Ref, entry lockfile.Entry, prov store.Provenance) error {
	from := "an unknown source, without a " + store.ProvenanceFile
	if prov.URL != "" {
		from = prov.URL + " at " + prov.Revision
	}
	return fmt.Errorf("%w: %s is installed from %s, this project wants it from %s"+
		"\nnote: pass --force to replace it, which changes what every project on this machine imports",
		ErrSourceConflict, ref, from, entry.URL)
}
