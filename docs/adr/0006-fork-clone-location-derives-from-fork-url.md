# The fork clone's location derives from the fork URL

A user may publish on behalf of more than one owner — their own fork for their
own packages, an organisation's fork for that organisation's. gotpm kept the
fork clone in one fixed directory and reused whatever clone it found there, so
changing `fork.url` left submissions being committed and pushed to the previous
fork with no warning. The default location of a fork clone is therefore derived
from the fork URL — host, owner, then repository, the layout cloned package
repositories already use — giving each fork its own clone and making the switch
between two forks need no manual step. A `fork.path` the user configured
explicitly still points where they set it.

## Considered Options

- **Verify the clone's origin and re-clone on mismatch.** Rejected as the
  mechanism: switching back and forth between two forks would re-clone every
  time, and a fork of `github.com/typst/packages` is the 1000+ package
  repository whose cost the sparse, blobless cloning in `internal/gitcli`
  exists to avoid. The origin check is kept as a guard — it reports a clone
  that belongs to another fork and stops, rather than re-cloning — because an
  explicitly configured `fork.path` gets no derivation and would otherwise keep
  the silent wrong push.
- **Document that the user deletes the clone before switching.** Rejected: a
  wrong push is silent, so documentation is not a safeguard.

## Consequences

- Publishing through several forks costs one clone per fork. That is affordable
  only because the clone is sparse and blobless, and acceptable only because a
  fork clone is a staging area holding no work that does not exist elsewhere.
- The fork clones live beside the remotes cache, not inside it. `gotpm cache
  clear` does not touch them: a clone holding a `gotpm publish --local` commit
  that has not been pushed yet is not state kept merely to avoid repeating
  work.
- The clone left at the old fixed location is moved to its derived location on
  the next publish, when its origin is the configured fork. One whose origin is
  a different fork is left alone and reported — that is a user who already hit
  this bug and has two forks tangled up, and choosing for them would be wrong.
  Every other way the move can fail is survivable and costs only a fresh clone,
  so none of it stops a publish. The move is an upgrade path, not behaviour:
  `migrateLegacyClone` and `legacyForkDirName` can be deleted once no supported
  version of gotpm still writes to `$DATA_DIR/fork`.
- Which fork a clone belongs to is decided by the canonical form
  `internal/resolve` already reduces a repository url to, the same form the
  lock file and provenance record. Deriving a second notion of "the same
  repository" here would let the two disagree about a url the user spelled
  unusually. The comparison folds case on a hosted fork, because a host is
  case-insensitive and the hosts these forks live on treat an owner and a
  repository that way too; a fork on this machine keeps its case, being named
  by a path.
