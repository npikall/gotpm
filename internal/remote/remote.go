package remote

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v6"
)

var ErrParseRepoName = errors.New("could not parse repository name")

func CloneRepo(remote, dest string) error {
	url := resolveCloneURL(remote)
	_, err := git.PlainClone(dest, &git.CloneOptions{URL: url, Progress: os.Stderr})
	if err != nil {
		return fmt.Errorf("cloning %q into %q: %w", remote, dest, err)
	}
	return nil
}

func RepoNameFromURL(remoteURL string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		return path.Base(u.Path), nil
	}
	if _, after, found := strings.Cut(trimmed, ":"); found {
		return path.Base(after), nil
	}
	return "", ErrParseRepoName
}

func resolveCloneURL(path string) string {
	if hasScheme(path) {
		return path
	}
	return "https://" + strings.TrimSuffix(path, ".git")
}

func hasScheme(path string) bool {
	return strings.Contains(path, "://")
}
