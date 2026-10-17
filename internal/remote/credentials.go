package remote

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

// credentialsFor asks git for the credentials the user keeps for an http(s)
// repository — whatever credential helper they set up, from the keychain to
// 'gh auth'. gotpm stores none of its own. Prompting is disabled, so a user
// without credentials for the host gets nothing rather than a question.
func credentialsFor(cloneURL string) (*githttp.BasicAuth, bool) {
	if !isHTTP(cloneURL) {
		return nil, false
	}
	out, err := credentialFill(cloneURL)
	if err != nil {
		return nil, false
	}
	return parseCredentials(out)
}

func isHTTP(cloneURL string) bool {
	u, err := url.Parse(cloneURL)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http")
}

// credentialFill runs 'git credential fill' for a url and returns its answer.
func credentialFill(cloneURL string) (*bytes.Buffer, error) {
	cmd := exec.CommandContext(context.Background(), "git", "credential", "fill")
	cmd.Stdin = strings.NewReader("url=" + cloneURL + "\n\n")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git credential fill: %w", err)
	}
	return &out, nil
}

func parseCredentials(out *bytes.Buffer) (*githttp.BasicAuth, bool) {
	auth := &githttp.BasicAuth{}
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		key, value, _ := strings.Cut(scanner.Text(), "=")
		switch key {
		case "username":
			auth.Username = value
		case "password":
			auth.Password = value
		}
	}
	return auth, auth.Password != ""
}

// needsCredentials reports whether a host refused a fetch the way it refuses
// one for a private repository.
func needsCredentials(err error) bool {
	return errors.Is(err, transport.ErrAuthenticationRequired) || errors.Is(err, transport.ErrRepositoryNotFound)
}
