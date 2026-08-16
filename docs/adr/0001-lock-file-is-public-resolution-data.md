# The lock file is public resolution data

A dependency is declared as `@gotpm/cetz:0.3.1`, which names a package but no
repository, and there is no gotpm registry that could map one to the other.
gotpm therefore resolves transitive dependencies by reading the `gotpm.lock` a
dependency committed beside its own `typst.toml` — so a package's lock is part
of what it publishes, not a private build artifact.

## Considered Options

- **A central registry mapping package refs to repositories.** Rejected: it is
  infrastructure to host and keep alive, and Typst Universe already is the
  registry for published packages — gotpm exists for the ones that are not
  there yet.
- **Repository URLs in `typst.toml` dependencies.** Rejected: the dependency
  string is what Typst source imports the package by, and it has to stay in the
  form the compiler understands.

## Consequences

- A package with dependencies and no committed lock cannot be depended on;
  resolution fails rather than guessing.
- The lock format is a compatibility surface. A change to it is read by other
  people's gotpm, not only by the one that wrote the file, which is why the
  format carries a schema version and refuses anything newer.
