---
icon: lucide/user
---

# Usage

## Depending on a repository

Typst resolves an import from one of two places: `@preview`, the official
registry, or a namespace somebody filled in by hand. Neither lets a document say
*"I depend on that repository at that version"* and have it come back on someone
else's machine. `add`, `sync` and `remove` add that missing layer.

A project using them keeps two files next to each other:

| File | What it owns |
| --- | --- |
| `typst.toml` | **which** packages the project wants, listed under `[tool.gotpm]` |
| `gotpm.lock` | **where** each one comes from, and the exact commit it was fetched at |

Packages are installed under the `@gotpm` namespace, which keeps them apart from
anything you placed in `@local` yourself.

### Commit `gotpm.lock`

`gotpm.lock` is not a cache. It belongs in version control, for two reasons:

1. It is what makes a checkout reproducible. `gotpm sync` installs the commit
   the lock names, not whatever the tag points at today, so a tag moved upstream
   cannot silently change your document.
2. It is the **only** record of where a dependency's own dependencies live. A
   dependency string such as `@gotpm/cetz:0.3.1` names a package, never a
   repository — so when your package is itself added by someone else, the lock
   you committed is what tells their gotpm where to fetch `cetz` from. A package
   that declares gotpm dependencies without committing its lock cannot be added
   at all.

### A worked example

Add a repository. Everything it depends on comes along:

```console
$ gotpm add github.com/user/cetz
info: added @gotpm/cetz:0.3.1 from github.com/user/cetz
info:   @gotpm/oxifmt:0.2.1 (via @gotpm/cetz:0.3.1)
```

Only the package you asked for is written to `typst.toml`; the rest are recorded
in the lock as transitive:

```toml
[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
```

while `gotpm.lock` gets the part that makes it reproducible — the repository, the
commit, and which package asked for it:

```json
{
  "version": 1,
  "packages": [
    {
      "import": "@gotpm/cetz:0.3.1",
      "name": "cetz",
      "version": "0.3.1",
      "namespace": "gotpm",
      "url": "github.com/user/cetz",
      "revision": "v0.3.1",
      "hash": "abc123def4567890abc123def4567890abc123de",
      "subdir": "",
      "direct": true,
      "required_by": null
    }
  ]
}
```

Import it the way the dependency list spells it:

```typst
#import "@gotpm/cetz:0.3.1": *
```

Anyone who clones the project gets the same packages from the two committed
files:

```console
$ git clone https://github.com/user/my-doc && cd my-doc
$ gotpm sync
info: installed @gotpm/oxifmt:0.2.1
info: installed @gotpm/cetz:0.3.1
```

In CI, use `--frozen`. A lock that disagrees with `typst.toml` then fails the run
instead of being quietly repaired, which is usually a change somebody forgot to
commit:

```console
$ gotpm sync --frozen
```

### Upgrading, and two versions at once

`add` never rewrites an existing entry, because that would invalidate the
`#import` statements already in your source. Adding a second version installs it
alongside the first and lists both:

```toml
[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
  "@gotpm/cetz:0.4.0",
]
```

Typst installs every version at its own path, so both remain importable. Once
your source no longer mentions the old one, take it out with
`gotpm remove @gotpm/cetz:0.3.1`.

## Project commands and standalone commands

The commands below fall into two groups, and the difference decides which flags
they take.

**Project commands** — `add`, `sync`, `remove` — take the current project as
their subject. They read `typst.toml` and `gotpm.lock`, and install or delete
the whole dependency graph those two describe. Run one outside a project and it
says so. `bump` and `locate` read the project as well, but change no dependency.

**Standalone commands** — `install`, `uninstall`, `list`, `check`, `publish`,
`init`, `update`, `cache`, `config`, `self` — need no project. `install` and
`uninstall` act on a single package version, which is why they are the only two
that accept `--install-dir`: that flag names a directory to receive one
package's files directly, and a dependency graph does not fit in one. A project
command always works on the package directory.

`$GOTPM_INSTALL_DIR` is the ambient form of `--install-dir` — the same
single-package destination, set once instead of typed each time. It is not a
setting for where gotpm keeps its data, and it is ignored by every command that
does not offer the flag. To move the package directory itself, layout and all,
set Typst's own `$TYPST_PACKAGE_PATH`; every command honours that.

## `add`

Add a repository as a dependency of the current project.

```console
$ gotpm help add
The package is installed under the @gotpm namespace, together with
everything it depends on, and recorded in two files next to typst.toml:

  typst.toml   gains the import under [tool.gotpm].dependencies
  gotpm.lock   pins every package to the exact commit it was fetched from

Commit both. gotpm.lock is what lets anyone who checks the project out run
'gotpm sync' and get the same packages, and it is the only place recording
where a dependency's own dependencies come from.

Without --rev the newest release tag is used, or the current HEAD when the
repository has no release tags.

USAGE
    gotpm add <repository> [--flags]

EXAMPLES
    gotpm add github.com/user/repo
    gotpm add github.com/user/repo -t v0.1.2
    gotpm add git@github.com:user/repo.git

FLAGS
    -f --force     Replace a package installed from a different repository.
    -h --help      Help for add
    -t --rev       The revision (hash or tag) to pin. Defaults to the newest release.
    -v --verbose   Enable verbose output
```

The package store is shared by every project on your machine. If
`@gotpm/cetz:0.3.1` is already installed from a different repository, `add`
refuses rather than overwrite it, since that would change what your other
projects import. `--force` overrides that, deliberately.

## `bump`

Bump the version of a Package.

```console
$ gotpm help bump
Use this command to change the version of the Package or to display it.

USAGE
    gotpm bump [--flags]

EXAMPLES
    gotpm bump major
    gotpm bump 0.1.2

FLAGS
    --dry-run Perform a dry-run
    -h --help Help for bump
    -s --show Show the version of the current package
    -v --verbose Print Debug Level Information
```

## `init`

Initialize a new minimal Typst Package.

```console
$ gotpm help init
Initialize a new minimal Typst Package

USAGE
    gotpm init [--flags]

FLAGS
    -h --help  Help for init
```

## `install`

Install a Package locally, such that the Typst compiler knows how to import it.

```console
$ gotpm help install
All files that are not specifically excluded get copied to
'$DATA_DIR/typst/packages', where the '$DATA_DIR' is dependend on
the machines operating system.

USAGE
    gotpm install [path] [--flags]

EXAMPLES
    gotpm install
    gotpm install --editable
    gotpm install --namespace preview
    gotpm install path/to/package/dir
    gotpm install path/to/package/dir -n preview

FLAGS
    -e --editable   If the installed package should be editable.
    -h --help       Help for install
    -n --namespace  The namespace in which the package should be available. (local)
    -V --verbose    Print Debug Level Information
```

## `list`

List all available Packages installed on your machine.

```console
$ gotpm help list
List all locally installed Packages

USAGE
    gotpm list [--flags]

EXAMPLES
    gotpm list

FLAGS
    -h --help     Help for list
    -V --verbose  Print Debug Level Information
```

## `locate`

Show every path and directory gotpm reads or writes.

```console
$ gotpm help locate
Show the paths and directories gotpm reads and writes.

Without a key, every path is listed, grouped by what it belongs to. The
project group is only shown when the working directory belongs to a typst
project.

With a key, only that path is printed, unstyled and on its own, so it can be
used directly in a shell.

Nothing is created: a path that does not exist yet is still where gotpm would
look for it.

USAGE
    gotpm locate [key] [--flags]

EXAMPLES
    # Show every path
    gotpm locate

    # Print one path, for use in a shell
    cd "$(gotpm locate packages)"

FLAGS
    -h --help     Help for locate
    -v --verbose  Enable verbose output
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

`$GOTPM_INSTALL_DIR` is reported deliberately, as a warning rather than a fact
about where dependencies go: a variable left set in your shell will send the
next `gotpm install` into a flat directory Typst cannot import from. `add`,
`sync` and `remove` ignore it and keep using the package directory, so the two
can disagree — that is what the annotation is for.

## `remove`

Remove a dependency from the current project.

```console
$ gotpm help remove
Takes the import string exactly as it appears in typst.toml, so the
version being removed is never in doubt.

The dependency is dropped from typst.toml and from gotpm.lock, along with
any package only it pulled in. The installed files are left in the package
store, because other projects on this machine may import the same version;
--prune deletes them too.

USAGE
    gotpm remove <@gotpm/name:version> [--flags]

EXAMPLES
    gotpm remove @gotpm/cetz:0.3.1
    gotpm rm @gotpm/cetz:0.3.1 --prune

FLAGS
    -h --help      Help for remove
    --prune        Delete the removed packages from the package store as well.
    -v --verbose   Enable verbose output
```

Removing a dependency also drops the transitive ones nothing else needs any
more. A package another dependency still requires stays:

```console
$ gotpm remove @gotpm/cetz:0.3.1
info: removed @gotpm/cetz:0.3.1
info:   no longer needed: @gotpm/oxifmt:0.2.1
info: the package files are still in the store; pass --prune to delete them
```

## `sync`

Install everything the current project depends on. This is what a fresh checkout
needs before it compiles.

```console
$ gotpm help sync
Reads typst.toml and gotpm.lock and makes the package store match them.
This is what a fresh checkout needs before it compiles.

Every package is installed at the commit gotpm.lock pins, not at whatever
its tag points at today. Lock entries that nothing in typst.toml requires
any more are dropped.

--frozen fails instead of rewriting gotpm.lock, which is what a CI job
wants: a lock that disagrees with typst.toml is a change somebody forgot
to commit.

USAGE
    gotpm sync [--flags]

EXAMPLES
    gotpm sync
    gotpm sync --frozen

FLAGS
    -f --force     Replace a package installed from a different repository.
    --frozen       Fail instead of updating gotpm.lock.
    -h --help      Help for sync
    -v --verbose   Enable verbose output
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

Uninstall a package, or a whole namespace.

```console
$ gotpm help uninstall
Removes a locally installed Typst package from the package directory.

Naming a namespace and nothing else removes the whole namespace, after asking
for confirmation. Adding a package, a version or --all narrows the removal back
to a package inside that namespace.

The package directory can be overridden via the --install-dir flag
or the GOTPM_INSTALL_DIR environment variable. The flag takes precedence.
A namespace cannot be removed from an overridden directory, which holds a
single package rather than a namespace layout.

USAGE
    gotpm uninstall [name] [--flags]

EXAMPLES
    # get package metadata from typst.toml
    gotpm uninstall
    gotpm uninstall foo

    # uninstall specific package from 'local' or 'preview'
    gotpm uninstall foo -V 0.1.2
    gotpm uninstall foo -V 0.1.2 -n preview

    # all versions of foo in namespace 'local' or 'preview'
    gotpm uninstall foo --all
    gotpm uninstall foo -n preview --all

    # the whole 'preview' namespace, with and without the prompt
    gotpm uninstall -n preview
    gotpm uninstall -n preview --yes

FLAGS
    --all           Uninstall all Packages from a given namespace or all versions of a package.
    --dry-run       Perform a dry run.
    -h --help       Help for uninstall
    --install-dir   Override the package directory (env: $GOTPM_INSTALL_DIR)
    -n --namespace  The namespace from which the package should be removed from. On its own, removes the whole namespace. (local)
    -v --verbose    Enable verbose output
    -V --version    The specific version of a package that should be removed.
    -y --yes        Skip the confirmation prompt when removing a namespace.
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
