package git

import (
	"os"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
)

// tokenEnvVars are the environment variables a token for an HTTPS remote is
// read from, in order of precedence.
var tokenEnvVars = []string{"GOTPM_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"}

// clientOptions returns the transport options for talking to rawURL.
//
// An SSH remote needs none: go-git falls back to the SSH agent on its own, so
// anyone who pushes with `git@` today keeps working with nothing configured.
// An HTTPS remote is the case gotpm has to answer for, because git's
// credential helpers - the macOS Keychain, `gh` - belong to the git binary and
// go-git cannot read them. A token from the environment stands in. Without
// one the request goes out unauthenticated, which is enough to read a public
// fork and not enough to push to it.
func clientOptions(rawURL string) []client.Option {
	if !isHTTP(rawURL) {
		return nil
	}
	token := envToken()
	if token == "" {
		return nil
	}
	// The token goes in as a password rather than a bearer token: that is the
	// form GitHub accepts for git over HTTPS, and the username is ignored.
	return []client.Option{client.WithHTTPAuth(&http.BasicAuth{Username: "git", Password: token})}
}

// HasHTTPToken reports whether an HTTPS remote could be authenticated.
func HasHTTPToken() bool {
	return envToken() != ""
}

// TokenEnvVars names the environment variables a token is read from, so a
// caller can say what to set.
func TokenEnvVars() []string {
	return append([]string(nil), tokenEnvVars...)
}

func envToken() string {
	for _, name := range tokenEnvVars {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token
		}
	}
	return ""
}

// isHTTP reports whether a remote URL is spoken over HTTP(S) rather than SSH.
func isHTTP(rawURL string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(rawURL))
	return strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")
}
