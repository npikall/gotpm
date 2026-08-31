---
title: Managing dependencies
icon: lucide/package
---

# Managing dependencies

This is the workflow for a project that *uses* packages: a thesis, a report, a
template repository. It covers depending on a git repository, reproducing that
dependency on another machine, and taking it away again.

Three commands do the work — `add`, `sync` and `remove` — and they all operate
on the same two files.

## Depending on a repository

Typst resolves an import from one of two places: `@preview`, the official
registry, or a namespace somebody filled in by hand. Neither lets a document say
*"I depend on that repository at that version"* and have it come back on someone
else's machine.

A project using gotpm keeps two files next to each other:

| File | What it owns |
| --- | --- |
| `typst.toml` | **which** packages the project wants, listed under `[tool.gotpm]` |
| `gotpm.lock` | **where** each one comes from, and the exact commit it was fetched at |

Packages are installed under the `@gotpm` namespace, which keeps them apart from
anything you placed in `@local` yourself.

## A worked example

Add a repository. Everything it depends on comes along:

```console
$ gotpm add github.com/user/cetz
info: added @gotpm/cetz:0.3.1 from github.com/user/cetz
info:   @gotpm/oxifmt:0.2.1 (via @gotpm/cetz:0.3.1)
```

Without `--rev`, the newest release tag is used — or the current `HEAD` when the
repository has no release tags. Pin something else explicitly:

```console
$ gotpm add github.com/user/cetz -t v0.3.0
$ gotpm add github.com/user/cetz -t 4f2a1c9
$ gotpm add git@github.com:user/private-pkg.git
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

## Commit `gotpm.lock`

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

## Reproducing it elsewhere

Anyone who clones the project gets the same packages from the two committed
files:

```console
$ git clone https://github.com/user/my-doc && cd my-doc
$ gotpm sync
info: installed @gotpm/oxifmt:0.2.1
info: installed @gotpm/cetz:0.3.1
```

`sync` also drops lock entries that nothing in `typst.toml` requires any more,
which is how a lock stays honest across branches.

### In CI

Use `--frozen`. A lock that disagrees with `typst.toml` then fails the run
instead of being quietly repaired, which is usually a change somebody forgot to
commit:

```yaml
- name: Install gotpm
  run: curl -sSfL https://github.com/npikall/gotpm/releases/latest/download/install.sh | sh

- name: Install dependencies
  run: gotpm sync --frozen

- name: Compile
  run: typst compile thesis.typ
```

## Upgrading, and two versions at once

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

Typst installs every version at its own path, so both remain importable. Migrate
your `#import` statements file by file, at your own pace. Once your source no
longer mentions the old one, take it out:

```console
$ gotpm remove @gotpm/cetz:0.3.1
```

## Removing a dependency

`remove` takes the import string exactly as it appears in `typst.toml`, so the
version being removed is never in doubt. It also drops the transitive
dependencies nothing else needs any more — a package another dependency still
requires stays:

```console
$ gotpm remove @gotpm/cetz:0.3.1
info: removed @gotpm/cetz:0.3.1
info:   no longer needed: @gotpm/oxifmt:0.2.1
info: the package files are still in the package directory; pass --prune to delete them
```

The files are left behind on purpose: the [package
directory](../concepts.md#the-package-directory) is shared by every project on
the machine, and another one may import the same version. `--prune` deletes them
anyway.

## Things that will bite you

### A dependency added to `typst.toml` by hand cannot be synced

An import statement names a package, never the repository it comes from, so only
you know where it should be fetched from:

```console
$ gotpm sync
ERROR

  Declared dependency is missing from the lock: @gotpm/cetz:0.3.1
  note: gotpm.lock records where a package comes from; run 'gotpm add <url>' to add it properly.
```

Use `gotpm add <repository>` instead. It writes both files.

### The same coordinate from a different repository

The package directory is shared by every project on your machine. If
`@gotpm/cetz:0.3.1` is already installed from a different repository, `add`
refuses rather than overwrite it, since that would change what your other
projects import. `--force` overrides that, deliberately.

### A dependency without a committed lock

If a repository you are adding declares gotpm dependencies but did not commit
its `gotpm.lock`, gotpm has no way to learn where those dependencies live, and
the add fails. That is a bug in the package you are adding — the fix is upstream
committing its lock.

## Checking that imports will resolve

`gotpm check` reads a `.typ` file and reports whether every package it imports
will resolve at compile time:

```console
$ gotpm check thesis.typ
```

A non-Universe package has to be present in the package directory; a `@preview`
package need only exist in the Universe index, because the compiler downloads it
and gotpm does not interfere.
