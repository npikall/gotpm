// Package compilers installs Typst compilers into the compiler directory, the
// machine-wide directory gotpm runs compilers from, one directory per version.
//
// A compiler is a Typst release. It is downloaded from a release source, its
// archive is checked against the digest the release publishes, and only its
// binary is kept. Installs are swapped into place in one rename, so a failed
// or concurrent install never leaves a half-installed compiler behind. Like
// the font directory, the compiler directory is not cache.
package compilers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrInvalidVersion      = errors.New("invalid compiler version")
	ErrNoRelease           = errors.New("no such compiler release")
	ErrUnsupportedPlatform = errors.New("typst ships no compiler for this platform")
	ErrChecksum            = errors.New("compiler archive does not match its digest")
	ErrNoDigest            = errors.New("compiler archive has no digest")
	ErrNotInstalled        = errors.New("compiler is not installed")
	ErrInvalidArchive      = errors.New("invalid compiler archive")
	ErrFailedRequest       = errors.New("http request failed")
)

// Source is somewhere Typst releases are obtained from.
type Source interface {
	// Releases lists every published release, pre-releases included.
	Releases(ctx context.Context) ([]Release, error)
	// Release returns the release of one version, or ErrNoRelease.
	Release(ctx context.Context, version string) (Release, error)
	// Download fetches one asset of a release.
	Download(ctx context.Context, asset Asset) ([]byte, error)
}

// Release is one published compiler.
type Release struct {
	// Version is the compiler's version, without the tag's "v".
	Version    string
	Prerelease bool
	Assets     []Asset
}

// Asset is one downloadable archive of a release.
type Asset struct {
	Name string
	URL  string
	// Digest is the archive's digest as GitHub publishes it, "sha256:<hex>",
	// or empty for older releases.
	Digest string
}

var (
	sourceMu sync.RWMutex
	source   Source = NewGitHub()
)

// Default is the source compilers are installed from.
func Default() Source { //nolint: ireturn // sources are interchangeable by design
	sourceMu.RLock()
	defer sourceMu.RUnlock()
	return source
}

// Use replaces the default source and returns a function restoring the
// previous one. It exists for tests.
func Use(s Source) func() {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	previous := source
	source = s
	return func() {
		sourceMu.Lock()
		defer sourceMu.Unlock()
		source = previous
	}
}

// Stable lists the releases that are not pre-releases, newest first.
func Stable(ctx context.Context, src Source) ([]Release, error) {
	releases, err := src.Releases(ctx)
	if err != nil {
		return nil, err
	}
	stable := slices.DeleteFunc(slices.Clone(releases), func(r Release) bool {
		return r.Prerelease || !isStable(r.Version)
	})
	slices.SortFunc(stable, func(a, b Release) int { return compareVersions(b.Version, a.Version) })
	return stable, nil
}

// Latest is the newest stable release.
func Latest(ctx context.Context, src Source) (Release, error) {
	stable, err := Stable(ctx, src)
	if err != nil {
		return Release{}, err
	}
	if len(stable) == 0 {
		return Release{}, fmt.Errorf("%w: no stable release published", ErrNoRelease)
	}
	return stable[0], nil
}

// NewestStable is the newest of versions that is not a pre-release, or ""
// when there is none.
func NewestStable(versions []string) string {
	newest := ""
	for _, version := range versions {
		if isStable(version) && (newest == "" || compareVersions(version, newest) > 0) {
			newest = version
		}
	}
	return newest
}

// isStable reports whether version is valid and carries no pre-release tag.
func isStable(version string) bool {
	return validVersion(version) == nil && !strings.Contains(version, "-")
}

// versionPattern matches a compiler version, "0.13.1" or "0.14.0-rc.1". It
// becomes a directory name and part of a URL, so nothing else is accepted.
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

func validVersion(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, version)
	}
	return nil
}

// compareVersions orders two valid versions; a pre-release sorts before the
// release it leads up to.
func compareVersions(a, b string) int {
	coreA, preA, _ := strings.Cut(a, "-")
	coreB, preB, _ := strings.Cut(b, "-")
	if c := slices.Compare(numbers(coreA), numbers(coreB)); c != 0 {
		return c
	}
	switch {
	case preA == preB:
		return 0
	case preA == "":
		return 1
	case preB == "":
		return -1
	default:
		return comparePre(preA, preB)
	}
}

// comparePre orders pre-release tags field by field, numeric fields as
// numbers, so "rc.2" sorts before "rc.10".
func comparePre(a, b string) int {
	fieldsA, fieldsB := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(fieldsA), len(fieldsB)) {
		x, errX := strconv.Atoi(fieldsA[i])
		y, errY := strconv.Atoi(fieldsB[i])
		c := strings.Compare(fieldsA[i], fieldsB[i])
		if errX == nil && errY == nil {
			c = cmp.Compare(x, y)
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(fieldsA), len(fieldsB))
}

func numbers(core string) []int {
	parts := strings.Split(core, ".")
	out := make([]int, len(parts))
	for i, part := range parts {
		out[i], _ = strconv.Atoi(part)
	}
	return out
}
