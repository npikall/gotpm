# The compiler pin lives in the manifest

`gotpm run` invokes a compiler gotpm installed itself, and inside a project it
uses the exact version the project pins. The `compiler` key Typst defines in
`[package]` is only a floor, so the exact version has to be recorded somewhere
else. It is recorded as `compiler-pin` in the manifest's `[tool.gotpm]`
section, beside the floor it has to satisfy, and `run` writes it there on its
own when it is missing or below the floor.

## Considered Options

- **Pin the compiler in the lock.** Rejected: the lock is public resolution
  data (ADR 0001), and a compiler pin resolves nothing for a dependent — only a
  project's own pin matters. Because gotpm decodes the lock leniently, adding
  it would also have to raise the lock schema version (as font pins did,
  ADR 0007), making every project that ever used `run` unreadable to dependents
  on an older gotpm, for data they ignore.
- **Pin the compiler in a `.typst-version` file beside the manifest**, the way
  uv pins Python. Rejected: a second file to commit and keep in step with the
  manifest, while the manifest already carries everything else the project
  decides for itself in `[tool.gotpm]`.

## Consequences

- `run` edits a hand-written file the first time it runs in a project. The edit
  is in place (ADR 0008), touches only that key, and is announced.
- Dependents read a dependency's manifest, but never its `compiler-pin`; only
  its `[package] compiler` floor counts toward theirs.
- `publish` copies the pin into the submission to the Typst Universe like the
  rest of `[tool.gotpm]`. The compiler ignores it.
