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

**Project commands** — [`add`](#add), [`sync`](#sync), [`remove`](#remove),
[`font add` and `font remove`](#font) —
take the current project as their subject. They read `typst.toml` and
`gotpm.lock`, and install or delete the whole dependency graph those two
describe. Run one outside a project and it says so. [`bump`](#bump) and
[`locate`](#locate) read the project as well, but change no dependency.

**Standalone commands** — [`install`](#install), [`uninstall`](#uninstall),
[`list`](#list), [`check`](#check), [`publish`](#publish), [`init`](#init),
[`update`](#update), [`cache`](#cache), [`config`](#config), [`self`](#self),
[`font install`, `font uninstall` and `font search`](#font) — need no project. `install` and `uninstall` act on a single package version,
which is why they are the only two that accept `--install-dir`: that flag names
a directory to receive one package's files directly, and a dependency graph does
not fit in one. A project command always works on the [package
directory](concepts.md#the-package-directory).

## Environment variables

| Variable | Effect |
| --- | --- |
| `$TYPST_PACKAGE_PATH` | Moves the package directory itself, layout and all. Honoured by every command. |
| `$GOTPM_INSTALL_DIR` | The ambient form of `--install-dir`: a single-package destination, set once instead of typed each time. Ignored by every command that does not offer the flag. |
| `$GITHUB_TOKEN` | Sent to the GitHub API, and only there, when the `font` commands look up a commit or the family list. Lifts the limit of 60 unauthenticated requests an hour. |

gotpm reads no font path of its own. Typst finds the fonts gotpm installs only
when `$TYPST_FONT_PATHS` points at the font directory — see [`font`](#font).

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

The fonts a dependency declares come along: `add` reads their pins from the
dependency's own `gotpm.lock`, records them in yours and installs them into the
[font directory](#font). They are pinned, not declared — `typst.toml` gains no
font. Where your lock already pins the family at another commit, that pin is
kept and the conflict is reported.

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

The fork clones `publish` stages submissions in are not cache either, and a
plain `gotpm cache clear` leaves them alone. `gotpm cache clear --forks` removes
them instead — and with them any `gotpm publish --local` commit not pushed yet.
A `fork.path` you configured is never touched.

The [font directory](#font) is not cache either. `gotpm cache clear --fonts`
removes every installed font family; `gotpm sync` brings back the ones a
project pins. A plain `clear` only drops the cached list `font search` reads.

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

## `font`

Install font families from [Google Fonts](https://github.com/google/fonts), and
pin the ones a project needs.

```console
--8<-- "docs/includes/cli/font.txt"
```

Fonts are kept in gotpm's font directory, one directory per family. Typst does
not look there on its own, so point `$TYPST_FONT_PATHS` at it once, e.g. in
`~/.bashrc`:

```sh
export TYPST_FONT_PATHS="$(gotpm locate fonts)"
```

| Command | Kind | Does |
| --- | --- | --- |
| `font install <name>` | standalone | Installs the newest commit of a family |
| `font uninstall <name>` | standalone | Deletes an installed family |
| `font search [query]` | standalone | Lists the families whose name contains the query |
| `font add <name>` | project | Installs a family, pins it in `gotpm.lock` and declares it in `typst.toml` |
| `font remove <name>` | project | Drops a family from `typst.toml` and `gotpm.lock`; the files stay |

A declared font is written under `[tool.gotpm]`, by the name Typst's `font`
setting uses:

```toml
[tool.gotpm]
fonts = [
  "Open Sans",
]
```

Its pin in `gotpm.lock` names the commit of the Google Fonts repository and the
SHA-256 of every file, so `gotpm sync` installs exactly the files that were
added, wherever it runs. A lock holding font pins is format version 2, which
gotpm older than font support refuses to read
([ADR 0007](adr/0007-font-pins-raise-the-lock-schema-version.md)); a lock
without them stays version 1.

Names match the way the repository lays out its directories, ignoring case,
spaces and punctuation: `"Open Sans"`, `"open sans"` and `opensans` are one
family.

The font directory is shared by every project on your machine, and holds one
copy of each family. Two projects pinning different commits of a family cannot
both be satisfied: the last `sync` wins and says what it replaced. A family
directory gotpm did not create — one without a `.gotpm.json` — is never
replaced or deleted without `--force`; `sync` skips it with a warning.

!!! warning "Variable fonts"
    The Google Fonts repository ships most families as variable fonts only, such
    as `Roboto[wdth,wght].ttf`. Typst renders a variable font at its default
    instance, so bold and light text come out regular. gotpm warns when a family
    has no static files.

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
| `fonts` | The fonts installed by `gotpm font install`, one directory per family |
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

Typst does not look in the `fonts` directory on its own. Point
`$TYPST_FONT_PATHS` at it, for example in `~/.bashrc`, and every installed
font is available to `typst compile`:

```sh
export TYPST_FONT_PATHS="$(gotpm locate fonts)"
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
should be fetched from. Use `gotpm add <repository>` instead. A font added to
`typst.toml` by hand is refused the same way; use `gotpm font add <name>`.

The fonts the lock pins — declared by the project or by its dependencies — are
installed into the [font directory](#font) at their pinned commits, each file
checked against its recorded digest.

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
