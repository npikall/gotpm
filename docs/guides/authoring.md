---
title: Authoring a package
icon: lucide/pen-tool
---

# Authoring a package

This is the workflow for a project that *is* a package: you are writing the
library, and you need it importable while you work on it.

## Scaffold

```console
$ gotpm init mypkg
info: initialize package "mypkg"
```

Two files, and nothing else:

```toml title="typst.toml"
[package]
name = "mypkg"
version = "0.1.0"
entrypoint = "lib.typ"
```

```typst title="lib.typ"
#let greet(name) = [Hello #name]
```

Called without a name, `gotpm init` scaffolds into the current directory and
takes the package name from it. The manifest is yours from here on — add
`authors`, `license`, `description` and the rest as you go; gotpm only ever adds
to what you wrote.

## Install it while you work on it

Typst can only import a package that sits in the [package
directory](../concepts.md#the-package-directory). `gotpm install` puts it there:

```console
$ gotpm install
```

Everything below the package root is copied, except what `.gitignore` or
`.typstignore` excludes, and except `.git`, `.gitignore` and `.typstignore`
themselves, which are never part of a package. The default namespace is
`@local`, so the result is importable as:

```typst
#import "@local/mypkg:0.1.0": *
```

### Editable installs

Copying is wrong for the package you are actively editing — every change would
need a reinstall. `--editable` symlinks the working tree into the package
directory instead:

```console
$ gotpm install --editable
```

Edits to the source are picked up on the next compile, with no reinstall step.
An editable install is a development aid: it lives under `@local`, and it is
never what a declared dependency resolves to.

### Installing someone else's repository

`install -r` installs straight from a git repository, without touching any
project files:

```console
$ gotpm install -r github.com/user/repo -t v0.1.2
```

This is the standalone counterpart of `add`. Nothing is declared, nothing is
locked, and nothing reproduces on another machine — reach for it to try a
package out, and for [`gotpm add`](dependencies.md) when the project should
depend on it.

### Installing into `@preview`

```console
$ gotpm install -n preview
```

Useful for testing what a Universe release will behave like locally, before
`gotpm publish` opens the pull request that puts it there for real. Be aware
that this shadows the real Universe package of the same coordinate for every
project on your machine, so clean it up afterwards.

## See what is installed

```console
$ gotpm list
```

Every package in the package directory, by namespace, name and version —
including the ones Typst downloaded itself and the ones you installed by hand.

## Bump the version

```console
$ gotpm bump patch      # 0.1.0 -> 0.1.1
$ gotpm bump minor      # 0.1.1 -> 0.2.0
$ gotpm bump major      # 0.2.0 -> 1.0.0
$ gotpm bump 0.4.2      # set explicitly
```

The version lives in the manifest, and that is what decides a package's
version — not the git tag you release it under. Two flags read without writing,
which is what you want in a release script:

```console
$ gotpm bump --show-current
0.1.1
$ gotpm bump minor --show-next
0.2.0
```

`--dry-run` prints what would change without touching the file, and `--indent`
keeps indentation in the rewritten `typst.toml`.

!!! warning "A released version is a promise"

    One coordinate names one content, for good. Once a version is published,
    changing what it contains breaks every document that pinned it. Bump
    instead.

## Keep Universe imports current

`gotpm update` rewrites the `@preview` imports in your source to the latest
published version of each package:

```console
$ gotpm update lib.typ          # one file, in place
$ gotpm update src/ -r          # a whole tree, recursively
$ cat lib.typ | gotpm update    # stdin to stdout
```

It fetches the Universe index once and resolves every import against it, so the
cost is one network request regardless of how many imports you have.
`--no-cache` skips the index cache when you need the truly current index.

This works on imported packages, not on declared dependencies, and it touches no
lock. For dependencies you added with `gotpm add`, [add the new version
alongside](dependencies.md#upgrading-and-two-versions-at-once) instead.

## Check before you ship

```console
$ gotpm check lib.typ
```

Reports whether every package the file imports will actually resolve when Typst
compiles it — a `@local` or `@gotpm` package has to be present in the package
directory, while a `@preview` package need only exist in the Universe index.

## Clean up

```console
$ gotpm uninstall                      # this package, version from typst.toml
$ gotpm uninstall mypkg -V 0.1.0       # one specific version
$ gotpm uninstall mypkg --all          # every version of it
$ gotpm uninstall -n preview           # the whole @preview namespace, after a prompt
```

Naming a namespace and nothing else removes the whole namespace, and asks first.
It refuses outright when there is no terminal to answer:

```console
$ gotpm uninstall -n preview
warning: will delete @preview: 2 packages, 3 versions
delete the whole namespace? [y/N]
```

`--dry-run` shows what would go, and `--yes` skips the prompt for scripts that
genuinely mean it.

## When it is ready

[Publishing to the Typst Universe](publishing.md) is the next step.
