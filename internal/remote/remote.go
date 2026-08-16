// Package remote deals with where a package's repository lives: parsing its
// URL and caching the clone of it on disk. Cloning itself belongs to
// internal/git.
package remote

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

var ErrParseRepoName = errors.New("could not parse repository name")

// OwnerFromURL returns the owner/organisation segment of a repository URL,
// e.g. "npikall" for "https://github.com/npikall/packages".
func OwnerFromURL(remoteURL string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		return path.Base(path.Dir(u.Path)), nil
	}
	if _, after, found := strings.Cut(trimmed, ":"); found {
		return path.Base(path.Dir(after)), nil
	}
	return "", ErrParseRepoName
}

func DefaultHTTPCloneURL(path string) string {
	if hasScheme(path) {
		return path
	}
	return "https://" + strings.TrimSuffix(path, ".git")
}

func hasScheme(path string) bool {
	return strings.Contains(path, "://")
}
