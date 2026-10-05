# Font pins raise the lock schema version

A project declares the font families it needs, and its lock pins each one —
the font source, the location inside it, the exact commit and a digest of every
file — beside the package pins, in a `fonts` array of its own. gotpm decodes
the lock leniently, so an older gotpm reading a lock with font pins would drop
the array without a word and, on its next write, erase them from a committed,
public file. Adding font pins therefore raises the lock schema version to 2,
which every older gotpm refuses to read.

## Considered Options

- **Add the `fonts` array under schema version 1.** Rejected: unknown fields
  are ignored on read, so the first `gotpm add` or `sync` run by an older gotpm
  would silently rewrite the lock without its font pins. A lock that loses data
  without a warning is worse than one that cannot be read.
- **Pin fonts as package entries with a marker field.** Rejected for the same
  reason — an older gotpm would misread a font pin as a package it cannot fetch
  — and because a font pin has no package ref, version or namespace to fill the
  package fields with.
- **Keep font pins in a second file beside the lock.** Rejected: dependencies
  resolve the fonts of their dependencies by reading the same public lock
  (ADR 0001), and two files that must be committed together drift apart.

## Consequences

- A package whose lock carries font pins cannot be depended on with a gotpm
  older than this change: resolution stops with an unsupported-format error
  rather than ignoring the fonts. That cost falls on other people, which is why
  it is recorded.
- A lock without font pins is still written as version 1, the lowest version
  that expresses it, so a project that declares no fonts — and every package
  depending only on such projects — stays readable by older gotpm.
- A font pin carries a `source` field although Google Fonts is the only font
  source implemented. A second source then needs no further schema change.
