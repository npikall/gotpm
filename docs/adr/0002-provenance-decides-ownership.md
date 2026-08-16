# Provenance decides ownership of an installed package

The package directory is shared: the Typst compiler defines it, the user
hand-installs into it, other tools write into it, and every project on the
machine imports from it. A coordinate such as `@gotpm/cetz:0.3.1` therefore
does not tell gotpm whether the directory holding it is the one a project asked
for. Each package gotpm installs records the repository and commit it came from
in a `.gotpm.json` beside its files, and that record — not the namespace, not
the lock — is what decides whether gotpm may replace or delete what is there.

## Considered Options

- **Trust the lock.** Rejected: the lock describes one project, while the
  directory is machine-global. A pin cannot speak for what another project
  installed under the same coordinate.
- **A machine-global database of what gotpm installed.** Rejected: it drifts
  out of sync with the directory it describes. A record kept inside the package
  survives the project that asked for it and disappears when the package is
  deleted, by any means.

## Consequences

- A package installed without provenance — by hand, by another tool, or by
  `gotpm install` — is never silently replaced. gotpm refuses and says so;
  overwriting it is the user's decision, taken with `--force`.
- An editable install has no provenance and is therefore foreign to `sync` by
  construction. That is the intended protection, not a gap: an editable install
  is a development aid under `@local`, never a declared dependency's target.
