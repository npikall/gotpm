# An install dir holds one package, so project commands never take one

An install dir receives one package's files directly, with no
namespace/name/version layout around them. A project command — `add`, `sync`,
`remove` — works on a dependency graph, and a graph has no single directory to
land in: every ref in it would resolve to the same path, so the packages would
collide with each other or overwrite one another. Project commands therefore
always work on the package directory, and only `install` and `uninstall`, which
act on one package version, accept an install dir.

`$GOTPM_INSTALL_DIR` is the ambient form of `--install-dir`, not a setting for
where gotpm keeps its data. It is refused wherever the flag is refused. The
variable that relocates the package directory is Typst's own
`$TYPST_PACKAGE_PATH`, which keeps the layout intact and is honoured everywhere.

## Considered Options

- **Refuse a graph in a flat store.** Rejected: it makes a flag exist in order
  to fail. `add` and `sync` install a graph by definition, so the flag could
  only ever be accepted for a single-package graph — a coincidence, not a use
  case.
- **Lay out namespace/name/version inside the install dir.** Rejected: that
  directory is then a package directory, and the flag is a second, worse
  spelling of `$TYPST_PACKAGE_PATH`.
- **Keep the flag and let `--force` overwrite.** Rejected: it produced a
  directory holding the files of several packages at once, and no error
  mentioned it.

## Consequences

- Project commands resolve the store through `store.OpenPackageDir`, which
  reads `$TYPST_PACKAGE_PATH` and ignores `$GOTPM_INSTALL_DIR`. `check` already
  did this for the same reason, and `list` now does too: it scans namespaces,
  which a directory holding one package does not have.
- Vendoring a whole dependency graph into a directory is not something gotpm
  does. `gotpm install --install-dir` vendors one package, deliberately.
- `gotpm locate` still reports `$GOTPM_INSTALL_DIR` when it is set, as a
  warning rather than a fact about where dependencies go: a variable that
  silently redirects `gotpm install` is worth seeing.
