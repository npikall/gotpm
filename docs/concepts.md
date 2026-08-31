---
title: Concepts
icon: lucide/book-open
---

# Concepts

gotpm is small, but it works on top of a model that Typst only half defines.
This page is that model in full. It is worth ten minutes if you intend to use
`add` and `sync` on anything you will still be maintaining next year.

## What Typst gives you, and what it leaves open

The Typst compiler resolves an import like `#import "@preview/cetz:0.3.1": *`
in one of two ways:

- the namespace is `@preview`, and the compiler downloads the package from the
  **Typst Universe** on its own;
- the namespace is anything else, and the compiler looks in a directory on your
  machine and expects the package to already be there.

That second branch is the gap. Typst says *where* a non-Universe package must
sit, and nothing about how it gets there, who put it there, or where it came
from. Copying a folder in by hand satisfies the compiler and satisfies nobody
else: a collaborator cloning your repository has no way to reproduce it.

gotpm is the machinery for that second branch.

## Packages and package refs

A **package** is a versioned, importable unit of Typst source, defined by the
`typst.toml` at its root. That file is the **manifest**: it holds what Typst
needs to compile the package and what the Universe needs to publish it. It is
yours — hand-written, hand-commented — and gotpm only ever adds to it.

A **package ref** is the coordinate you import a package by:

```typst
#import "@gotpm/cetz:0.3.1": *
//        ^      ^     ^
//        |      |     └── version
//        |      └──────── name
//        └─────────────── namespace
```

A package ref names a package. It carries no content, no source and, crucially,
**no repository**. Nothing in `@gotpm/cetz:0.3.1` says where cetz comes from.
That single fact shapes most of gotpm's design, and the [lock file](#the-lock)
is the answer to it.

!!! note "A version is a promise"

    The version in the manifest is what decides a package's version — not the
    git tag it was released under. One coordinate names one content, for good.
    Publishing different content under a version already released is an
    authoring error, and not one gotpm tries to resolve for you.

## Namespaces

The namespace is the first segment of a package ref. It scopes where the
compiler resolves the import from, and nothing more. gotpm works with three:

| Namespace | Who fills it | What it means |
| --- | --- | --- |
| `@preview` | The Typst Universe | The compiler downloads it. gotpm reads such imports and can rewrite their versions, but never installs them. |
| `@local` | You | Typst's own convention for hand-installed packages. `gotpm install` targets it by default. |
| `@gotpm` | gotpm | Where a declared dependency of a project is installed. Kept separate so `sync` never fights with what you placed in `@local` yourself. |

A namespace says nothing about ownership. `@gotpm/cetz:0.3.1` being present
does not by itself mean gotpm installed it — for that, see
[provenance](#provenance).

## The package directory

The **package directory** is the directory the Typst compiler resolves packages
from. It lives under your platform's data directory (`$DATA_DIR/typst/packages`)
and is laid out by namespace, name and version:

```
packages/
├── local/
│   └── mypkg/
│       └── 0.1.0/
│           ├── typst.toml
│           └── lib.typ
└── gotpm/
    └── cetz/
        ├── 0.3.1/
        └── 0.4.0/
```

Three things follow from that layout, and they explain a good deal of gotpm's
behaviour:

**It is machine-global, not per-project.** Every project on your machine imports
out of the same directory. Two projects depending on `@gotpm/cetz:0.3.1` share
one installed copy. This is why `gotpm remove` does not delete files by default:
the version you no longer need may be the version another project still
compiles against.

**Every version lives at its own path.** Two versions of one package coexist
happily, which is what makes upgrading incremental rather than a flag day.

**gotpm is one writer among several.** Typst writes there. You write there. Other
tools write there. gotpm therefore takes care to touch only what is its own.

Set `$TYPST_PACKAGE_PATH` to move the whole thing. The layout inside is
unchanged and Typst imports from it exactly the same way; every gotpm command
honours it.

### The install dir is a different thing

`--install-dir`, and the `$GOTPM_INSTALL_DIR` environment variable behind it,
look similar and are not:

| | Package directory | Install dir |
| --- | --- | --- |
| Set by | `$TYPST_PACKAGE_PATH` | `--install-dir` / `$GOTPM_INSTALL_DIR` |
| Layout | `namespace/name/version/` | the package's files, directly |
| Typst imports from it | yes | no |
| gotpm scans it | yes | no |
| Holds | many packages | one package |

An install dir is an *output destination* — vendoring one package into a build
directory, or inspecting what an install would produce. Because it holds one
package, only `install` and `uninstall` accept it. `add`, `sync` and `remove`
work on a whole dependency graph, which does not fit, so they ignore it
entirely and always use the package directory.

!!! warning "A stray `$GOTPM_INSTALL_DIR` is a trap"

    Left set in your shell, it sends the next `gotpm install` into a flat
    directory Typst cannot import from, while `add` and `sync` carry on using
    the package directory. `gotpm locate` annotates the `packages` path when the
    variable is set, precisely so the two cannot silently disagree.

## Provenance

When gotpm installs a package it records, inside the installed package, the
repository and the exact commit it was built from. That record is its
**provenance**, and it is the only thing that decides whether an installed
package is gotpm's to replace or remove.

The consequence you will actually meet: if `@gotpm/cetz:0.3.1` is already
installed from a *different* repository, `gotpm add` refuses rather than
overwrite it. Overwriting would change what every other project on the machine
imports under that coordinate. `--force` overrides it, deliberately and
loudly.

## The manifest and the lock

A project using gotpm dependencies keeps two files side by side:

| File | What it owns |
| --- | --- |
| `typst.toml` | **which** packages the project wants, listed under `[tool.gotpm]` |
| `gotpm.lock` | **where** each one comes from, and the exact commit it was fetched at |

The split is the whole trick. A manifest entry is a package ref, which cannot
name a repository. The lock supplies what the ref cannot.

### The lock

A **lock** names, for every package the project depends on directly or
transitively, the repository and the exact commit it came from. One entry is a
**pin**.

```json
{
  "import": "@gotpm/cetz:0.3.1",
  "url": "github.com/user/cetz",
  "revision": "v0.3.1",
  "hash": "abc123def4567890abc123def4567890abc123de",
  "direct": true,
  "required_by": null
}
```

`gotpm.lock` is **not a cache** and belongs in version control, for two
reasons:

1. **It is what makes a checkout reproducible.** `sync` installs the commit the
   lock names, not whatever the tag points at today.
2. **It is the only record of where a dependency's own dependencies live.**
   There is no gotpm registry that could map `@gotpm/cetz:0.3.1` back to a
   repository. So when your package is added by someone else, the lock you
   committed is what tells *their* gotpm where to fetch cetz's dependencies
   from. A package that declares gotpm dependencies without committing its lock
   cannot be added at all.

### Pruning and drift

**Pruning** is dropping the pins the lock no longer reaches from any declared
dependency, so removing one dependency also clears the transitive ones only it
pulled in. It happens on its own, and it never deletes files.

**Drift** is a pin whose revision no longer points at the commit that was
pinned — an upstream tag that was moved. The pin still decides what gets
installed; drift is reported, never acted on.

!!! note "`remove --prune` means something else"

    Confusingly, `gotpm remove --prune` uninstalls the removed packages from the
    package directory. It is unrelated to lock pruning, and the flag is known to
    be misnamed.

## Declared dependencies vs. imported packages

These are two different lists, and gotpm never assumes they agree:

- a **declared dependency** is a package the manifest declares and the lock
  pins. gotpm fetches and installs it. Always imported under `@gotpm`.
- an **imported package** is a package a `.typ` file imports. Mostly a Universe
  package, which the compiler fetches by itself.

A file may import a package the project never declared, and a declared
dependency no file imports is still installed. They meet only in the package
directory, where a declared dependency has to be present for an import of it to
resolve. `gotpm check` is the command that reports whether they do.

## The cache

gotpm keeps two things purely to avoid repeating work: the repositories it has
cloned, and the Typst Universe version index it has fetched. That is the whole
of the **cache**, `gotpm cache clear` empties it, and deleting it loses nothing.

The package directory is never cache.

## Project commands and standalone commands

Every gotpm command is one or the other, and the difference decides which flags
it takes.

**Project commands** — `add`, `sync`, `remove` — take the current project as
their subject. They read `typst.toml` and `gotpm.lock` and install or delete the
whole dependency graph those two describe. Run one outside a project and it says
so. Because a graph is many packages, a project command always works on the
package directory and never takes an install dir. (`bump` and `locate` read the
project too, but change no dependency.)

**Standalone commands** — `install`, `uninstall`, `list`, `check`, `publish`,
`init`, `update`, `cache`, `config`, `self` — need no project, so the directory
they operate on can be named on the command line.

## Where to go next

- [Managing dependencies](guides/dependencies.md) — the `add`/`sync`/`remove`
  workflow end to end
- [Authoring a package](guides/authoring.md) — scaffold, install, iterate, bump
- [Publishing](guides/publishing.md) — getting a release into the Typst Universe
- [Command reference](reference.md) — every command and flag
