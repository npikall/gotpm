package resolve

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

var ErrInvalidRepoURL = errors.New("not a valid repository url")

const minSegments = 3

const fileScheme = "file://"

// PathSeparator splits a repository from the package path inside it, as in
// "github.com/a/mono//pkg".
const PathSeparator = "//"

// Source is a repository in the two forms gotpm needs it: the canonical name it
// is recorded under, and the address git is pointed at.
type Source struct {
	// Canonical is the scheme-less form of the repository, e.g.
	// "github.com/a/cetz", so the same repository is recognised however the
	// user happened to spell it. Messages about the repository itself — its
	// clone, tags and revisions — name it; messages about the package name
	// the whole Source, which adds the package path.
	Canonical string
	// CloneURL is what git is handed. A user who asked for a repository over
	// ssh gets ssh, so their existing credentials keep working.
	CloneURL string
	// Path is the package path: where the package's root sits inside the
	// repository, empty for the repository root.
	Path string
}

// String is the source as a user writes it: the canonical repository, followed
// by the package path when there is one. It is what goes into gotpm.lock and
// into a package's provenance.
func (s Source) String() string {
	return JoinPath(s.Canonical, s.Path)
}

// PackageDir is the package's root inside a working tree of the repository.
func (s Source) PackageDir(worktree string) string {
	return filepath.Join(worktree, filepath.FromSlash(s.Path))
}

// JoinPath appends a package path to a repository.
func JoinPath(repository, packagePath string) string {
	if packagePath == "" {
		return repository
	}
	return repository + PathSeparator + packagePath
}

// Normalize reads a repository argument in any of the forms a user is likely to
// write it: a bare path, an https url, or scp-style ssh.
func Normalize(raw string) (Source, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Source{}, fmt.Errorf("%w: empty", ErrInvalidRepoURL)
	}

	trimmed, packagePath, err := splitPackagePath(trimmed)
	if err != nil {
		return Source{}, fmt.Errorf("%w %q: %w", ErrInvalidRepoURL, raw, err)
	}

	if IsLocal(trimmed) {
		local := strings.TrimSuffix(strings.TrimSuffix(trimmed, "/"), ".git")
		return Source{Canonical: local, CloneURL: local, Path: packagePath}, nil
	}

	canonical, err := canonicalize(trimmed)
	if err != nil {
		return Source{}, err
	}
	if err := validate(canonical, raw); err != nil {
		return Source{}, err
	}

	cloneURL := trimmed
	if !hasScheme(trimmed) && !isSCPLike(trimmed) {
		cloneURL = "https://" + canonical
	}
	return Source{Canonical: canonical, CloneURL: cloneURL, Path: packagePath}, nil
}

var errPathOutsideRepository = errors.New("package path leaves the repository")

// splitPackagePath cuts the package path off a repository argument.
func splitPackagePath(raw string) (string, string, error) {
	start := separatorSearchStart(raw)
	i := strings.Index(raw[start:], PathSeparator)
	if i < 0 {
		return raw, "", nil
	}
	repository := raw[:start+i]
	cleaned := path.Clean(strings.Trim(raw[start+i:], "/"))
	if cleaned == "." {
		return repository, "", nil
	}
	if !filepath.IsLocal(filepath.FromSlash(cleaned)) {
		return "", "", errPathOutsideRepository
	}
	return repository, cleaned, nil
}

// separatorSearchStart skips a scheme's "://" and the character after it, so
// neither that nor the leading slash of an absolute file:// path reads as a
// separator.
func separatorSearchStart(raw string) int {
	start := 1
	if i := strings.Index(raw, "://"); i >= 0 {
		start = i + len("://") + 1
	}
	return min(start, len(raw))
}

func canonicalize(raw string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSuffix(raw, "/"), ".git")

	if host, path, ok := cutSCPLike(trimmed); ok {
		return host + "/" + strings.Trim(path, "/"), nil
	}

	if !hasScheme(trimmed) {
		return strings.Trim(trimmed, "/"), nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w %q: %w", ErrInvalidRepoURL, raw, err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%w %q: missing host", ErrInvalidRepoURL, raw)
	}
	return parsed.Host + "/" + strings.Trim(parsed.Path, "/"), nil
}

func validate(canonical, raw string) error {
	segments := strings.Split(canonical, "/")
	if len(segments) < minSegments {
		return fmt.Errorf("%w %q: expected at least host/owner/repository", ErrInvalidRepoURL, raw)
	}
	if !strings.Contains(segments[0], ".") {
		return fmt.Errorf("%w %q: %q is not a host", ErrInvalidRepoURL, raw, segments[0])
	}
	if slices.Contains(segments, "") {
		return fmt.Errorf("%w %q: empty path segment", ErrInvalidRepoURL, raw)
	}
	return nil
}

func cutSCPLike(raw string) (string, string, bool) {
	if hasScheme(raw) {
		return "", "", false
	}
	before, after, found := strings.Cut(raw, ":")
	if !found || after == "" || strings.Contains(after, ":") {
		return "", "", false
	}
	if _, hostPart, hasUser := strings.Cut(before, "@"); hasUser {
		return hostPart, after, hostPart != ""
	}
	return before, after, true
}

func isSCPLike(raw string) bool {
	_, _, ok := cutSCPLike(raw)
	return ok
}

func hasScheme(raw string) bool {
	return strings.Contains(raw, "://")
}

// IsLocal reports whether a source is a repository on this machine rather than
// a hosted one.
func IsLocal(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), fileScheme)
}
