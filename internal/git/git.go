// Package git performs the git operations publishing needs through go-git,
// so that gotpm does not require a git binary on the user's machine.
//
// The fork clone it manages has no worktree. Publishing only ever writes one
// directory into the fork, so the file contents of typst/packages' other 1500
// packages are never needed, and a commit can be assembled from tree objects
// alone. Avoiding a checkout is also what makes a pure go-git implementation
// possible: go-git can ask a server for a partial clone but has no promisor
// support, so it cannot lazily fetch a blob that the clone's filter left
// behind, and any checkout of a filtered clone fails with "object not found".
//
// The clone is therefore blobless, single-branch and depth-1 - roughly
//
//	git clone --depth 1 --no-checkout --filter=blob:none
//
// which for typst/packages is one round trip and a few megabytes. A tree:0
// filter would transfer less still, but every tree on the path down to the
// package would then have to be fetched one round trip at a time, because
// GitHub caps the tree filter at depth 0 ("tree filter allows max depth 0")
// and so cannot return the whole spine in one request.
package git

import (
	"errors"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/npikall/gotpm/internal/paths"
)

const (
	// RemoteName is the only remote a fork clone has.
	RemoteName = "origin"
	// MainBranch is the branch pull requests are opened against.
	MainBranch = "main"
)

// ErrNoMainBranch reports a clone with no resolvable origin/main, which means
// its initial clone never completed.
var ErrNoMainBranch = errors.New("clone has no origin/main")

// Fork is a clone of a fork of typst/packages.
type Fork struct {
	repo *gogit.Repository
	path string
}

// Clone creates the fork clone at path. The clone is blobless, shallow,
// single-branch and has no worktree; see the package documentation for why.
func Clone(url, path string) (*Fork, error) {
	var repo *gogit.Repository
	err := withDegradedTransport(func(filter packp.Filter, depth int) error {
		// A refused option leaves the half-written clone behind, which the
		// retry would then trip over as an existing repository.
		if err := paths.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		var err error
		repo, err = gogit.PlainClone(path, &gogit.CloneOptions{
			URL:   url,
			Depth: depth,
			// Naming the branch explicitly is what makes the clone record
			// refs/remotes/origin/main. Left to its default, go-git tracks the
			// remote's HEAD under refs/remotes/origin/HEAD instead, and every
			// later lookup of origin/main comes up empty.
			ReferenceName: localRef(MainBranch),
			SingleBranch:  true,
			NoCheckout:    true,
			Tags:          plumbing.NoTags,
			Filter:        filter,
		})
		if err != nil {
			return fmt.Errorf("cloning %q: %w", url, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fork := &Fork{repo: repo, path: path}
	if err := fork.recordPartialClone(); err != nil {
		return nil, err
	}
	return fork, nil
}

// Open opens an existing fork clone.
func Open(path string) (*Fork, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening clone at %q: %w", path, err)
	}
	return &Fork{repo: repo, path: path}, nil
}

// Path returns the clone's location on disk.
func (f *Fork) Path() string {
	return f.path
}

// HasMain reports whether origin/main resolves, i.e. whether the clone's
// initial fetch actually completed.
func (f *Fork) HasMain() bool {
	_, err := f.repo.Reference(remoteRef(MainBranch), true)
	return err == nil
}

// recordPartialClone marks the clone as a partial clone of origin, the way
// canonical git does. go-git never writes these keys itself, which leaves a
// filtered clone looking merely corrupt: without them `git` run by hand in the
// fork directory reports the filtered-out objects as missing instead of
// fetching them on demand.
func (f *Fork) recordPartialClone() error {
	cfg, err := f.repo.Config()
	if err != nil {
		return fmt.Errorf("reading clone config: %w", err)
	}
	if cfg.Raw == nil {
		// Nothing to attach the keys to. The clone still works for gotpm; only
		// hand-run git in that directory suffers, so this is not fatal.
		return nil
	}
	cfg.Raw.Section("extensions").SetOption("partialClone", RemoteName)
	remote := cfg.Raw.Section("remote").Subsection(RemoteName)
	remote.SetOption("promisor", "true")
	remote.SetOption("partialclonefilter", string(packp.FilterBlobNone()))

	if err := f.repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("recording partial-clone config: %w", err)
	}
	return nil
}

// withDegradedTransport runs fn with the most economical transport options the
// server will accept, retrying with less as it refuses them.
//
// Partial clone and shallow fetch are both optional server extensions. GitHub
// implements both, but go-git's own server - which backs the file:// URLs the
// tests use, and which some self-hosted forks sit behind - implements neither,
// and rejects the request outright rather than ignoring the option.
func withDegradedTransport(fn func(filter packp.Filter, depth int) error) error {
	filter, depth := packp.FilterBlobNone(), 1
	for {
		err := fn(filter, depth)
		switch {
		case errors.Is(err, transport.ErrFilterNotSupported) && filter != "":
			filter = ""
		case errors.Is(err, transport.ErrShallowNotSupported) && depth != 0:
			depth = 0
		default:
			return err
		}
	}
}

// localRef names a branch in the clone.
func localRef(branch string) plumbing.ReferenceName {
	return plumbing.NewBranchReferenceName(branch)
}

// remoteRef names the fork's copy of a branch.
func remoteRef(branch string) plumbing.ReferenceName {
	return plumbing.NewRemoteReferenceName(RemoteName, branch)
}

// refSpec is the refspec that updates the fork's copy of branch. Clone leaves
// the remote configured with a single-branch refspec covering main only, so
// every other branch has to be named explicitly.
func refSpec(branch string) config.RefSpec {
	return config.RefSpec(fmt.Sprintf("+%s:%s", localRef(branch), remoteRef(branch)))
}
