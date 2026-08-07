// Package resolve turns a repository argument into an installable package: it
// clones the repository, decides which revision to use, checks it out and reads
// the manifest found there.
//
// It is the piece install and add have in common. What they do with the result
// differs — install places it in the store and forgets about it, add records it
// in the project — but getting from "github.com/a/cetz" to a directory holding
// a known version at a known commit is the same problem.
package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/semver"
)

var ErrNotAPackage = errors.New("not a typst package")

const (
	// Latest asks for the newest release rather than a particular revision.
	Latest = "latest"
	// headRevision is what a repository without release tags is pinned to.
	headRevision = "HEAD"
	// tagPrefix is the "v" release tags conventionally carry.
	tagPrefix = "v"
)

// Request is a repository and the revision of it that is wanted. An empty
// Revision, or Latest, asks for the newest release tag.
type Request struct {
	URL      string
	Revision string
}

// Resolved is a repository checked out at a known commit, with the manifest of
// the package it holds.
type Resolved struct {
	Source Source
	// Revision is what was actually used: the tag that was picked, whatever
	// the caller asked for, or HEAD for a repository without release tags.
	Revision string
	Hash     string
	// Dir is the working tree of the cached clone.
	Dir      string
	Manifest *manifest.Manifest
}

// Ref returns the package reference this resolves to in a namespace. The
// version always comes from the manifest, never from the tag: typst requires
// the version in the install path to match the one the package declares.
func (r *Resolved) Ref(namespace string) (pkg.Ref, error) {
	return pkg.New(namespace, r.Manifest.Package.Name, r.Manifest.Package.Version)
}

// Resolve fetches a repository and reads the package in it.
func Resolve(req Request, logger *log.Logger) (*Resolved, error) {
	src, err := Normalize(req.URL)
	if err != nil {
		return nil, err
	}

	clone, err := remote.EnsureClone(src.Canonical, src.CloneURL)
	if err != nil {
		return nil, fmt.Errorf("%w%s", err, subdirHint(src))
	}
	defer clone.Repo.Close() //nolint: errcheck
	logger.Debug("resolved remote", "url", src.Canonical, "path", clone.Dir, "cloned", clone.Cloned)

	revision, err := pickRevision(clone.Repo, req.Revision, src, logger)
	if err != nil {
		return nil, err
	}
	hash, err := remote.ResolveHash(clone.Repo, revision)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src.Canonical, err)
	}
	// Checking out the commit rather than the revision keeps an install
	// reproducible when a tag is later moved.
	if err := remote.CheckoutRevision(clone.Repo, hash); err != nil {
		return nil, fmt.Errorf("checking out %s of %s: %w", revision, src.Canonical, err)
	}
	logger.Debug("checked out", "url", src.Canonical, "revision", revision, "hash", hash)

	m, err := loadRootManifest(clone.Dir, src)
	if err != nil {
		return nil, err
	}
	warnOnVersionMismatch(m, revision, src, logger)

	return &Resolved{Source: src, Revision: revision, Hash: hash, Dir: clone.Dir, Manifest: m}, nil
}

// pickRevision decides what to check out: what the caller asked for, or the
// newest release when they did not ask for anything in particular.
func pickRevision(repo *git.Repository, requested string, src Source, logger *log.Logger) (string, error) {
	if requested != "" && requested != Latest {
		return requested, nil
	}

	tags, err := remote.Tags(repo)
	if err != nil {
		return "", err
	}
	if tag, ok := LatestStableTag(tags); ok {
		logger.Debug("picked latest release", "url", src.Canonical, "tag", tag)
		return tag, nil
	}

	logger.Warn("no release tags, pinning the current HEAD", "url", src.Canonical)
	return headRevision, nil
}

// LatestStableTag picks the newest release from a list of tag names. Tags that
// are not a plain three-part version are ignored, which leaves out both
// prereleases such as "v0.3.0-rc1" and moving tags such as "nightly".
func LatestStableTag(tags []string) (string, bool) {
	var (
		bestName    string
		bestVersion *semver.Version
	)
	for _, tag := range tags {
		version, err := semver.Parse(strings.TrimPrefix(tag, tagPrefix))
		if err != nil {
			continue
		}
		if bestVersion == nil || version.Compare(bestVersion) > 0 {
			bestName, bestVersion = tag, version
		}
	}
	return bestName, bestVersion != nil
}

// loadRootManifest reads the manifest a repository must have at its root.
//
// manifest.LoadFrom is deliberately not used: it searches parent directories,
// which from inside the clone cache would reach directories that have nothing
// to do with this repository.
func loadRootManifest(dir string, src Source) (*manifest.Manifest, error) {
	m, err := manifest.LoadFile(filepath.Join(dir, manifest.FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s has no %s at its root%s",
			ErrNotAPackage, src.Canonical, manifest.FileName, subdirHint(src))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src.Canonical, err)
	}
	return m, nil
}

// warnOnVersionMismatch reports a release tag that disagrees with the version
// the package declares. The manifest wins, because that is the version typst
// expects to find the package installed under.
func warnOnVersionMismatch(m *manifest.Manifest, revision string, src Source, logger *log.Logger) {
	tagged := strings.TrimPrefix(revision, tagPrefix)
	if !semver.IsValidVersion(tagged) || tagged == m.Package.Version {
		return
	}
	logger.Warn("tag disagrees with the version in the manifest, using the manifest",
		"url", src.Canonical, "tag", revision, "manifest", m.Package.Version)
}

// subdirHint explains the most likely reason a deep path failed, since gotpm
// cannot yet tell a repository in a group from a package in a subdirectory.
func subdirHint(src Source) string {
	if IsLocal(src.Canonical) || strings.Count(src.Canonical, "/") < minSegments {
		return ""
	}
	return "\nnote: if this path points inside a repository, packages in a subdirectory are not supported yet"
}
