# A diverged package branch resets onto the fork

A package branch of the fork clone can move on in two places at once: on the
fork, when the author edits the staged `typst.toml` on GitHub because the
Universe maintainers asked for a change, and locally, when a `gotpm publish`
run commits into the clone. Neither copy is then a fast-forward of the other.

`publish` resets the local branch onto `origin/<branch>` and carries on. The
package files it is about to copy in are re-staged and re-committed on top, so
the reset costs a commit but no content.

## Considered Options

- **Refuse and print the commands to reconcile the two by hand.** What gotpm
  did until now. Rejected: it is how you treat a repository whose commits are
  the only copy of something, and the fork clone is not that. Every commit on a
  package branch is a copy of the package's working tree at
  `packages/preview/<name>/`, which the same run regenerates. Hand-merging two
  copies of generated content is work that cannot pay off.
- **Merge or rebase onto the fork.** Rejected: both can stop on a conflict and
  leave the fork clone mid-operation, which is a state the user now has to get
  out of before they can publish again — the failure mode the refusal already
  had, with an unfinished operation added to it.
- **Reset unconditionally, without the fast-forward first.** Rejected: a branch
  merely ahead of the fork is the ordinary result of `gotpm publish --local`,
  and its unpushed commit is deliberate. Only divergence resets.
- **Treat every refused fast-forward as divergence.** Rejected: `git merge
  --ff-only` also fails for reasons that have nothing to do with the two
  branches — a busy index, a git that cannot run at all — and answering an
  unexplained failure by discarding commits acts on a problem that was never
  diagnosed. Divergence is therefore established on its own, by asking whether
  either ref contains the other; anything else is reported as the error it is.

## Consequences

- An edit made on the fork survives; a local commit that diverged from it does
  not. That is the right way round: the fork is where a submission is reviewed,
  and the local commit's content comes back with the copy that follows.
- The reset is reported through the logger at warn level, so it is visible at
  the default verbosity rather than being a silent rewrite.
- Anything a user hand-wrote directly in the fork clone and committed there is
  lost on divergence. Edits belong in the package's own repository, which the
  fork clone explicitly is not.
