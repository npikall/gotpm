package git

import (
	"errors"
	"fmt"
	"io"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// ErrNoCredentials is returned when a remote refused gotpm because it had
// nothing to authenticate with.
var ErrNoCredentials = errors.New("no credentials for the remote")

// Push sends branch to origin, updating the remote branch of the same name.
//
// What is sent is worked out by walking the new commit against what the remote
// already has, and that walk compares subtrees by hash rather than reading
// through them. A commit that changed one package directory therefore sends
// that directory, and the objects a blobless clone never fetched are never
// asked for.
func (r *Repo) Push(branch string) error {
	url, err := r.originURL()
	if err != nil {
		return err
	}

	ref := plumbing.NewBranchReferenceName(branch)
	err = r.repo.Push(&gogit.PushOptions{
		RemoteName:    gogit.DefaultRemoteName,
		RefSpecs:      []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", ref, ref))},
		ClientOptions: clientOptions(url),
		Progress:      io.Discard,
	})
	if err == nil || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil
	}
	if isAuthError(err) {
		return fmt.Errorf("%w: %w\n%s", ErrNoCredentials, err, credentialHint(url))
	}
	return fmt.Errorf("pushing %q to %q: %w", branch, url, err)
}

func isAuthError(err error) bool {
	return errors.Is(err, transport.ErrAuthenticationRequired) ||
		errors.Is(err, transport.ErrAuthorizationFailed) ||
		errors.Is(err, transport.ErrRepositoryNotFound)
}

// credentialHint says what to do about a remote that would not let gotpm in.
// It is worth spelling out because the same fork usually pushes fine from the
// user's own shell: git reads a credential helper that gotpm cannot.
func credentialHint(url string) string {
	if !isHTTP(url) {
		return "gotpm authenticates SSH remotes through the SSH agent. Check that a key is loaded:\n" +
			"  ssh-add -l"
	}
	if HasHTTPToken() {
		return "The token gotpm found does not grant push access to this fork."
	}
	return "gotpm cannot read git's credential helpers, so an HTTPS remote needs a token:\n" +
		"  export " + TokenEnvVars()[0] + "=<token with repo scope>\n" +
		"Or point the fork at SSH, which needs no token:\n" +
		"  gotpm config set fork.url git@github.com:<owner>/packages\n" +
		"gotpm reads a token from " + strings.Join(TokenEnvVars(), ", ") + "."
}
