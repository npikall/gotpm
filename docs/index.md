---
title: GoTPM
icon: lucide/rocket
---
![Image title](assets/logo/gotpm-mark.svg){ align=right }

# A package manager for Typst

Typst can compile a package and it can download one from the Typst Universe.
Everything between those two points is left to you: getting a package you are
writing into a place the compiler looks, depending on a repository that is not
in the Universe, and reproducing either of those on someone else's machine.

**gotpm** is a single Go binary that fills in that gap.

```console
$ gotpm add github.com/user/cetz
info: added @gotpm/cetz:0.3.1 from github.com/user/cetz
info:   @gotpm/oxifmt:0.2.1 (via @gotpm/cetz:0.3.1)
```

<div class="grid cards" markdown>

- :lucide-download: **Install it**

    ---

    One binary, no runtime. Homebrew, a shell script, `go install`, or a
    release download.

    [:octicons-arrow-right-24: Installation](installation.md)

- :lucide-book-open: **Understand the model**

    ---

    Package refs, namespaces, the package directory, and why the lock file is
    committed.

    [:octicons-arrow-right-24: Concepts](concepts.md)

- :lucide-package: **Depend on a repository**

    ---

    `add`, `sync` and `remove`: git repositories as dependencies, pinned to a
    commit and reproducible anywhere.

    [:octicons-arrow-right-24: Managing dependencies](guides/dependencies.md)

- :lucide-pen-tool: **Write and ship a package**

    ---

    Scaffold, install locally while you work on it, bump the version, and open
    a pull request against the Typst Universe.

    [:octicons-arrow-right-24: Authoring a package](guides/authoring.md)

</div>

## What it is for

gotpm serves two people, and most projects are both at different moments.

### The document author

You are writing a thesis, a report, a template. You want to use a package that
lives in a git repository — because it is not published to the Universe, because
you need a version that is not released yet, or because it is yours and internal.

Typst gives you two ways to do that today: import from `@preview`, or copy files
into `@local` by hand. Neither survives a `git clone` on another machine.

gotpm adds the missing layer. `gotpm add <repository>` records the dependency in
`typst.toml` and pins the exact commit in a committed `gotpm.lock`, so a fresh
checkout runs `gotpm sync` and gets byte-identical packages — including the
dependencies of your dependencies.

### The package author

You are writing the package itself. You need it importable while you work on it,
you need its version to move in one step, and eventually you need it in the
Typst Universe.

`gotpm install --editable` symlinks the working tree into the package directory,
so `#import "@local/mypkg:0.1.0"` picks up your edits with no reinstall.
`gotpm bump minor` moves the version in the manifest. `gotpm publish` commits the
package into your fork of `typst/packages` on its own branch, ready for a pull
request.

## Why it might suit you

| | |
| --- | --- |
| **Reproducible by default** | `gotpm.lock` pins a commit hash, not a tag. A tag moved upstream cannot silently change your document. |
| **Transitive dependencies work** | A dependency's own lock is read to resolve what *it* needs — there is no registry in between, and none is required. |
| **Nothing to install first** | A single static binary. Only `gotpm publish` needs a system `git`; everything else, remote clones included, is pure Go. |
| **It shares nicely** | The package directory belongs to Typst, not to gotpm. Packages you installed by hand are left alone, and gotpm refuses to overwrite what it did not install. |
| **CI-shaped** | `gotpm sync --frozen` fails on a lock that disagrees with the manifest instead of quietly repairing it. |

[Compare it with the alternatives](comparison.md) if you are already using
`utpm` or `typship`.

<img class="shadow" src="assets/casette.gif" alt="gotpm in a terminal">
