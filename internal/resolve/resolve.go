// Package resolve turns a repository argument into an installable package: it
// clones the repository, decides which revision to use, checks it out and reads
// the manifest found there.
//
// depgraph.Walk calls it once per node of a dependency graph — the piece add's
// and install --remote's discovery share, whether the node is the root or a
// transitive dependency. Getting from "github.com/a/cetz" to a directory
// holding a known version at a known commit is the same problem either way.
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
	Latest       = "latest"
	headRevision = "HEAD"
	tagPrefix    = "v"
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
	// Dir is the package's root inside the working tree of the cached clone:
	// the directory at the package path, holding the manifest and the lock.
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
		return nil, fmt.Errorf("%w%s", err, packagePathHint(src))
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
	if err := remote.CheckoutRevision(clone.Repo, hash); err != nil {
		return nil, fmt.Errorf("checking out %s of %s: %w", revision, src.Canonical, err)
	}
	logger.Debug("checked out", "url", src.Canonical, "revision", revision, "hash", hash)

	packageDir := src.PackageDir(clone.Dir)
	m, err := loadPackageManifest(packageDir, src)
	if err != nil {
		return nil, err
	}
	warnOnVersionMismatch(m, revision, src, logger)

	return &Resolved{Source: src, Revision: revision, Hash: hash, Dir: packageDir, Manifest: m}, nil
}

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

func loadPackageManifest(packageDir string, src Source) (*manifest.Manifest, error) {
	m, err := manifest.LoadFile(filepath.Join(packageDir, manifest.FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, missingManifestError(src)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src, err)
	}
	return m, nil
}

func missingManifestError(src Source) error {
	if src.Path != "" {
		return fmt.Errorf("%w: %s has no %s", ErrNotAPackage, src, manifest.FileName)
	}
	return fmt.Errorf("%w: %s has no %s at its root%s",
		ErrNotAPackage, src, manifest.FileName, packagePathNote(src.Canonical))
}

func warnOnVersionMismatch(m *manifest.Manifest, revision string, src Source, logger *log.Logger) {
	tagged := strings.TrimPrefix(revision, tagPrefix)
	if !semver.IsValidVersion(tagged) || tagged == m.Package.Version {
		return
	}
	logger.Warn("tag disagrees with the version in the manifest, using the manifest",
		"url", src, "tag", revision, "manifest", m.Package.Version)
}

func packagePathHint(src Source) string {
	if IsLocal(src.Canonical) || src.Path != "" || strings.Count(src.Canonical, "/") < minSegments {
		return ""
	}
	return packagePathNote(firstSegments(src.Canonical, minSegments))
}

func packagePathNote(repository string) string {
	return "\nnote: name a package in a subdirectory by its package path, e.g. " +
		JoinPath(repository, "path/to/package")
}

func firstSegments(canonical string, n int) string {
	return strings.Join(strings.SplitN(canonical, "/", n+1)[:n], "/")
}
