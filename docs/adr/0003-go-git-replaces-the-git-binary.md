# go-git replaces the git binary

gotpm shelled out to `git` for the whole publish flow, because a blobless
partial clone of the Typst Universe package repository combined with a sparse
checkout appeared to need real git: go-git cannot lazily fetch objects a
server-side filter excluded, so any operation reaching for a filtered-out blob
fails where real git's promisor-remote support would fetch it on demand. The
publish flow never needs one, and gotpm now talks to a repository only through
go-git. The Fork Clone is cloned with `filter=blob:none`, the subtree of the
package being published is fetched explicitly by its tree hash, and the
worktree is scoped to that subtree; go-git's sparse checkout flags every path
outside the scope `SkipWorktree` in the index, so the commit carries the full
tree and touches no blob it does not have.

## Considered Options

- **Keep the git binary for publish only.** Rejected: it is the operation most
  likely to run where git is not installed, and a dependency needed by one
  command is a dependency of the whole tool.
- **Stop before committing and print the git commands to run.** Rejected:
  publish exists to get a submission staged and pushed, and handing back a
  command is handing back the work.

## Consequences

- gotpm owns push credentials. It tries the SSH agent first, which go-git
  reaches on its own, and falls back to a token from the environment for HTTPS
  remotes. Git's credential helpers — the macOS Keychain, `gh` — are a git
  binary feature and are unreachable, so a user who pushes over HTTPS today
  with a Keychain-stored token must supply a token or switch the fork to an
  SSH URL.
- Commits are no longer signed. gitcli committed through git and inherited the
  user's `commit.gpgsign`; go-git signs only when handed a signer.
- The Fork Clone is blobless but no longer shallow. Blobs are what make the
  repository large, and shallow history is where go-git is weakest, so the
  depth limit bought little and cost every fetch, merge and push an edge case.
- Nothing outside `internal/git` imports go-git, so the credential handling has
  one home.
