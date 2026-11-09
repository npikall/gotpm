---
title: Running the compiler
icon: lucide/play
---

# Running the compiler

A Typst project compiles only with a compiler that is new enough, and Typst's
0.x releases break things between minor versions. `gotpm run` takes the choice
of compiler off your hands: it stands in for `typst`, picks the compiler the
project needs, installs it when it is missing, and hands every
argument on to it unchanged.

```console
$ gotpm run compile main.typ
$ gotpm run watch main.typ
```

Anything you would type after `typst` goes after `gotpm run`. The exit code,
the output and Ctrl-C during `watch` are the compiler's own.

## Where the compiler comes from

gotpm runs only compilers it installed itself, into its **compiler directory**,
one directory per version. A `typst` on your `$PATH` is never used, so which
compiler runs does not depend on how your shell is set up.

`gotpm locate compilers` prints where it is. Like the font directory, the
compiler directory is not cache: `gotpm cache clear` leaves it alone.

`run` touches the network only when the compiler it needs is not installed yet.
It never checks for newer releases on its own, so once a compiler is installed
it starts instantly and works offline. A download is checked against the
SHA-256 digest Typst's GitHub release publishes for it; a mismatch installs
nothing. Old releases that publish no digest are installed over HTTPS with a
warning.

## Inside a project

Inside a project — the working directory or one of its parents holds a
`typst.toml` — the **compiler pin** decides the compiler. It is the exact
version the project is compiled with, kept in gotpm's section of the manifest:

```toml title="typst.toml"
[package]
name = "mypkg"
version = "0.1.0"
entrypoint = "lib.typ"
compiler = "0.13.0"

[tool.gotpm]
compiler-pin = "0.13.1"
```

Commit it, and your collaborators and CI compile with the same compiler you do.
A pin that satisfies the project's floor always wins over newer releases, so a
build never changes underneath you.

### The floor

The `compiler` key Typst defines in `[package]` is the project's **compiler
requirement**: the lowest compiler it compiles with. A floor, not an exact
version — any newer compiler satisfies it.

The floor `run` works with is the highest compiler requirement among the
project and its declared dependencies, read from the dependencies' installed
manifests. A dependency needing a newer compiler than you do raises your floor.
A declared dependency that is not installed yet is not counted, and `run` warns
about it — run `gotpm sync` after a fresh checkout.
A dependency's own compiler pin is never read: its author's choice of compiler
does not dictate yours.

### When `run` writes the compiler pin

When the project pins no compiler yet, the first `run` picks one and pins it:
the newest compiler you have installed that satisfies the floor, or else the
latest stable release that does.

```console
$ gotpm run compile main.typ
info: pinned compiler 0.13.1 in ~/mypkg/typst.toml
```

When you raise the floor above the pin — or add a dependency that does — the
next `run` pins again, and says which requirement made it:

```console
$ gotpm run compile main.typ
info: raised the compiler pin from 0.12.0 to 0.13.1 in ~/mypkg/typst.toml: 0.12.0 is below the compiler requirement 0.13.0 of @gotpm/cetz:0.3.1
```

The edit touches only the `compiler-pin` key and keeps the comments and layout
of your manifest. A project without a compiler requirement still gets a pin;
it is chosen as if outside a project.

## Outside a project

A loose `.typ` file compiles with the newest stable compiler you have
installed, or with the latest stable release when you have none. Nothing is
written anywhere but the compiler directory.

```console
$ cd ~/notes
$ gotpm run compile notes.typ
```

## Moving the compiler pin

`gotpm compiler pin` sets the compiler pin by hand, and installs the compiler when it is
missing, so the next `run` does not stall on a download:

```console
$ gotpm compiler pin 0.14.0
info: moved the compiler pin from 0.13.1 to 0.14.0 in ~/mypkg/typst.toml
```

Without a version it pins the latest stable release, which makes upgrading one
command. A version below the project's floor is refused.

Pre-releases are never chosen on their own: "latest" always means a stable
release. To test against an upcoming Typst, name the pre-release:

```console
$ gotpm compiler pin 0.14.0-rc.1
```

## One run with another compiler

`--typst` runs exactly the compiler you name for one invocation, installed
first when missing. It writes nothing; the compiler pin stays as it is. This is the way
to compare output across versions:

```console
$ gotpm run --typst 0.12.0 compile main.typ out-0.12.pdf
$ gotpm run compile main.typ out-pinned.pdf
```

## Flags and arguments

gotpm's own flags end at the first argument that is not one of them, so
everything from `compile` or `watch` onwards reaches the compiler untouched —
`gotpm run compile --help` shows Typst's help, not gotpm's. To hand the
compiler a flag in first position, put `--` before it:

```console
$ gotpm run -- --version
```

## Fonts

`run` appends gotpm's [font directory](../reference.md#font) to
`$TYPST_FONT_PATHS`, after any paths you set yourself, so the font families
gotpm installed are found without any shell setup. Your own paths keep working.

`run` never syncs: it fetches no package and no font family. After a fresh
checkout, run `gotpm sync` first.

## Seeing and freeing installed compilers

```console
$ gotpm compiler list
0.12.0
0.13.1 (pinned)
0.14.0 (newest)
```

The newest stable compiler is marked, and so, inside a project, is the compiler
pin —
also when it is not installed.

```console
$ gotpm compiler uninstall 0.12.0
info: uninstalled compiler 0.12.0
```

Uninstalling a compiler some project pins is allowed. The compiler directory
knows nothing about projects; the next `run` in that project installs it again.

## Where to go next

- [Concepts](../concepts.md#compilers) — the compiler pin and floor in the
  wider model
- [Command reference](../reference.md#run) — every flag of `run` and `compiler`
