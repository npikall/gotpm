# A private pin is just a pin

A dependency fetched from a private repository is locked exactly like any other:
repository, commit and package path, in the same committed, public lock and the
same schema. Nothing in the lock says the repository is private, because gotpm
only finds that out by trying to clone it — privacy belongs to whoever is
cloning, not to the pin. A lock is recreatable by anyone who can clone the
repositories it pins; anyone who cannot gets an unreachable-pin warning, the
same one they would get for a public repository that has since disappeared.

## Considered Options

- **Mark private pins in the lock** (`private = true`). Rejected: it would claim
  something about the repository that is only true for some readers, and it
  raises the lock schema for every other gotpm reading it (see ADR 0007).
- **A separate, uncommitted lock for private pins.** Rejected: the lock is the
  resolution data dependents read (ADR 0001); splitting it off makes the
  project's own lock incomplete for everyone, including users who do have
  access.

## Consequences

- A private repository's URL is visible in any lock that pins it, directly or
  transitively. Committing that lock is the author's choice.
- Fetching past an unreachable pin warns and skips it rather than failing, for
  `install --remote`, `add` and `sync` alike, so a lock with private pins in it
  stays usable by people without access.
- Supporting `add` for private repositories later needs no change to the lock.
