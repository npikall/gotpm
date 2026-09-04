---
title: Publishing
icon: lucide/send
---

# Publishing to the Typst Universe

The Typst Universe is a git repository: [`typst/packages`][packages]. Publishing
a package means opening a pull request that adds
`packages/preview/<name>/<version>/` to it. There is no upload endpoint and no
account to register.

`gotpm publish` does the tedious half of that — the fork, the branch, the
correct destination directory, the commit — and hands you a `gh pr create`
command for the rest. gotpm never talks to GitHub itself.

  [packages]: https://github.com/typst/packages

## One-time setup

Fork [`typst/packages`][packages] on GitHub yourself; gotpm never creates a
fork. Then tell gotpm where it is:

```console
$ gotpm config set fork.url https://github.com/you/packages
```

Optionally, choose where the clone of that fork lives. Unset, it is derived
from `fork.url` — `forks/<host>/<owner>/<repo>` inside gotpm's data directory:

```console
$ gotpm config set fork.path ~/src/typst-packages
$ gotpm config list
```

That clone is a **fork clone**: gotpm's staging area, holding no work that does
not exist elsewhere. Delete it whenever you like; the next publish clones it
again. It is checked out sparsely, scoped to the package directory being
published, so cloning the whole Universe is not the cost it sounds like.

If you publish on behalf of more than one organisation, each fork gets its own
clone, and switching `fork.url` is enough to switch between them: the default
location follows the url. A `fork.path` you set yourself does not follow it, so
switch the two together — gotpm refuses to publish into a clone of another fork
rather than pushing your submission to the wrong owner.

## Publishing a version

From the package's working tree:

```console
$ gotpm publish
info: committed "release: mypkg 0.1.0" on branch mypkg-0.1.0
info: pushed mypkg-0.1.0. Open a PR with:
gh pr create --repo typst/packages --base main --head you:mypkg-0.1.0 --draft --title "release: mypkg 0.1.0"
```

What happened, in order:

1. The manifest is located, and the **package root** — the directory holding
   `typst.toml`, not necessarily the one you ran the command in — becomes the
   source.
2. The fork clone is created if missing, and its `origin/main` is fetched, so a
   clone you made months ago does not cut a branch from a stale base.
3. A branch named `<name>-<version>` is checked out, tracking the fork's branch
   of that name if it already exists.
4. The package files are copied into `packages/preview/<name>/<version>/`,
   honouring `.gitignore` and `.typstignore`, with `.git`, `.gitignore` and
   `.typstignore` themselves left out.
5. The result is committed and pushed to your fork.

Then run the `gh pr create` line it printed. A **submission** — one package
version on one branch of your fork — is as far as gotpm goes; publication is the
Universe maintainers merging it.

### Reviewing before you push

`--local` stops after the commit and prints the push command instead of running
it:

```console
$ gotpm publish --local
info: committed "release: mypkg 0.1.0" on branch mypkg-0.1.0
info: push it when you are ready:
git -C /home/you/.local/share/gotpm/forks/github.com/you/packages push origin mypkg-0.1.0
```

Useful the first time, when you want to see exactly what landed in the fork
clone before anything becomes public.

### Fix-ups

Publishing the same version again — because a reviewer asked for a change —
checks out the existing branch and commits on top of it, so the open pull
request updates rather than a second one appearing. For a fix-up run the commit
message is taken from your package repository's own `HEAD` commit, so the reason
for the change carries over instead of a second identical `release:` line.

Override it whenever you want:

```console
$ gotpm publish -m "fix: entrypoint path in the manifest"
```

!!! warning "Bump instead of re-releasing"

    Fix-ups are for a submission that has not been merged yet. Once a version is
    in the Universe, its content is fixed: one coordinate names one content, for
    good. Publish a fix as a new version.

## A release checklist

```console
$ gotpm bump minor          # move the version in typst.toml
$ gotpm check lib.typ       # every import resolves
$ gotpm install -n preview  # try the Universe coordinate locally
$ gotpm publish             # stage the submission and push
```

Do not forget the parts gotpm cannot do for you: the Universe requires a
`description`, a `license` and an `authors` list in the manifest, and it expects
a `README.md` beside it. Read [the submission guidelines][guidelines] once
before your first package.

  [guidelines]: https://github.com/typst/packages#submission-guidelines

## Requirements

`gotpm publish` is the only command that shells out to a system `git` binary —
it needs sparse checkout and commit signing, which the pure-Go git library
cannot do. Every other command, remote installs included, needs no `git` on your
`$PATH`.
