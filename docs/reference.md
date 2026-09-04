---
title: Command reference
icon: lucide/terminal
---

# Command reference

Every command, with the help text generated from the binary itself. `gotpm help
<command>` prints the same thing in your terminal.

`-v/--verbose` is accepted everywhere and is repeatable: `-vv` is louder than
`-v`.

## Project commands and standalone commands

The commands fall into two groups, and the difference decides which flags they
take.

**Project commands** — [`add`](#add), [`sync`](#sync), [`remove`](#remove) —
take the current project as their subject. They read `typst.toml` and
`gotpm.lock`, and install or delete the whole dependency graph those two
describe. Run one outside a project and it says so. [`bump`](#bump) and
[`locate`](#locate) read the project as well, but change no dependency.

**Standalone commands** — [`install`](#install), [`uninstall`](#uninstall),
[`list`](#list), [`check`](#check), [`publish`](#publish), [`init`](#init),
[`update`](#update), [`cache`](#cache), [`config`](#config), [`self`](#self) —
need no project. `install` and `uninstall` act on a single package version,
which is why they are the only two that accept `--install-dir`: that flag names
a directory to receive one package's files directly, and a dependency graph does
not fit in one. A project command always works on the [package
directory](concepts.md#the-package-directory).

## Environment variables

| Variable | Effect |
| --- | --- |
| `$TYPST_PACKAGE_PATH` | Moves the package directory itself, layout and all. Honoured by every command. |
| `$GOTPM_INSTALL_DIR` | The ambient form of `--install-dir`: a single-package destination, set once instead of typed each time. Ignored by every command that does not offer the flag. |

`$GOTPM_INSTALL_DIR` is not a setting for where gotpm keeps its data. Left set
in a shell, it sends the next `gotpm install` into a flat directory Typst cannot
import from, while `add`, `sync` and `remove` carry on using the package
directory — which is why [`locate`](#locate) reports it as a warning rather than
a fact.

## `add`

Add a repository as a dependency of the current project.

```console
--8<-- "docs/includes/cli/add.txt"
```

The package directory is shared by every project on your machine. If
`@gotpm/cetz:0.3.1` is already installed from a different repository, `add`
refuses rather than overwrite it, since that would change what your other
projects import. `--force` overrides that, deliberately.

[:octicons-arrow-right-24: Managing dependencies](guides/dependencies.md)

## `bump`

Change the version of the current package, or print it.

```console
--8<-- "docs/includes/cli/bump.txt"
```

## `cache`

Manage the repositories gotpm has cloned and the Universe index it has fetched.

```console
--8<-- "docs/includes/cli/cache.txt"
```

The cache exists only to avoid repeating work. Deleting it loses nothing; the
[package directory](concepts.md#the-package-directory) is never cache.

## `check`

Report whether every package a Typst file imports will resolve when it is
compiled.

```console
--8<-- "docs/includes/cli/check.txt"
```

A package outside the Typst Universe has to be present in the package
directory; a Universe package need only exist in the index, because the compiler
downloads it and gotpm does not interfere.

## `config`

Read and write gotpm's own configuration, stored as TOML in the user config
directory.

```console
--8<-- "docs/includes/cli/config.txt"
```

Both keys concern [publishing](guides/publishing.md): `fork.url` is required
before `gotpm publish` will run, and `fork.path` defaults to a location derived
from `fork.url` — `forks/<host>/<owner>/<repo>` inside gotpm's data directory —
so each fork gets a clone of its own.

## `init`

Scaffold a minimal Typst package: a `typst.toml` and a `lib.typ`.

```console
--8<-- "docs/includes/cli/init.txt"
```

## `install`

Install a package into the package directory, so the Typst compiler can find it.

```console
--8<-- "docs/includes/cli/install.txt"
```

[:octicons-arrow-right-24: Authoring a package](guides/authoring.md)

## `list`

List every package installed on this machine.

```console
--8<-- "docs/includes/cli/list.txt"
```

## `locate`

Show every path and directory gotpm reads or writes.

```console
--8<-- "docs/includes/cli/locate.txt"
```

### Keys

| Key | Points at |
| --- | --- |
| `packages` | The Typst package directory packages are installed into |
| `data-dir` | gotpm's own data directory |
| `config-dir` | gotpm's own config directory |
| `config` | `config.toml`, gotpm's configuration file |
| `index` | `index-cache.json`, the cached package index |
| `remotes` | The cache of cloned remote repositories |
| `root` | The directory of the current project |
| `manifest` | The project's `typst.toml` |
| `lock` | The project's `gotpm.lock` |

`root`, `manifest` and `lock` describe the project the working directory
belongs to. Without one, they are left out of the listing, and asking for them
by name is an error.

The `packages` path follows the same overrides `gotpm install` does —
`$GOTPM_INSTALL_DIR` first, then `$TYPST_PACKAGE_PATH` — and the listing notes
which one applied:

```console
$ GOTPM_INSTALL_DIR=/tmp/scratch gotpm locate
Typst
  packages   /tmp/scratch (via $GOTPM_INSTALL_DIR)
...
```

## `publish`

Stage a package version in a fork of the Typst Universe repository, ready for a
pull request.

```console
--8<-- "docs/includes/cli/publish.txt"
```

[:octicons-arrow-right-24: Publishing](guides/publishing.md)

## `remove`

Remove a dependency from the current project.

```console
--8<-- "docs/includes/cli/remove.txt"
```

Removing a dependency also drops the transitive ones nothing else needs any
more. A package another dependency still requires stays:

```console
$ gotpm remove @gotpm/cetz:0.3.1
info: removed @gotpm/cetz:0.3.1
info:   no longer needed: @gotpm/oxifmt:0.2.1
info: the package files are still in the package directory; pass --prune to delete them
```

## `self`

Inspect or update the gotpm binary itself.

```console
--8<-- "docs/includes/cli/self.txt"
```

`gotpm self update` replaces the running binary with the latest GitHub release.
Installs made through a package manager are better updated through it instead.

## `sync`

Install everything the current project depends on. This is what a fresh checkout
needs before it compiles.

```console
--8<-- "docs/includes/cli/sync.txt"
```

A dependency added to `typst.toml` by hand cannot be synced: an import statement
names a package, never the repository it comes from, so only you know where it
should be fetched from. Use `gotpm add <repository>` instead.

```console
$ gotpm sync
ERROR

  Declared dependency is missing from the lock: @gotpm/cetz:0.3.1
  note: gotpm.lock records where a package comes from; run 'gotpm add <url>' to add it properly.
```

## `uninstall`

Delete an installed package from the package directory — one version, every
version of a package, or a whole namespace.

```console
--8<-- "docs/includes/cli/uninstall.txt"
```

Removing a namespace asks before it deletes anything, and refuses outright when
there is no terminal to answer:

```console
$ gotpm uninstall -n preview
warning: will delete @preview: 2 packages, 3 versions
delete the whole namespace? [y/N]

$ gotpm uninstall -n preview --dry-run
warning: dryrun would delete "~/.local/share/typst/packages/preview": 2 packages, 3 versions
```

## `update`

Rewrite the `@preview` imports of a source file to the latest published version
of each package.

```console
--8<-- "docs/includes/cli/update.txt"
```

It works on imported packages, not on declared dependencies, and touches no
lock. The Universe index is fetched once and every import resolved against it,
so the cost is one network request regardless of how many imports a file has.
