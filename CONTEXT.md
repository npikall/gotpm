# gotpm

A package manager for Typst. It makes a package the Typst compiler can find,
records where each dependency came from, and gets a finished package into the
Typst Universe.

## Language

### Packages

**Package**:
A versioned, importable unit of Typst source, defined by the `typst.toml` at
its root.
_Avoid_: library, module, dependency (when the source is meant)

**Package Ref**:
The coordinate a package is imported by — `@namespace/name:version`. It names a
package; it carries no source, no content and no repository.
_Avoid_: import string, package name, coordinate, ID

**Version**:
The version a package declares in its manifest. Tags do not decide it, and it
is a promise: one coordinate names one content, for good. Publishing different
content under a version already released is an authoring error gotpm does not
resolve.
_Avoid_: tag, release

**Manifest**:
The `typst.toml` that defines a package — what Typst needs to compile it, what
the Universe needs to publish it, and the dependencies the project declares.
It is written and commented by hand and stays the author's, whatever gotpm
records in it.
_Avoid_: config, package file, metadata

**Working Tree**:
A directory on disk holding a package as it stands right now, with no commit
behind it. What `install` and `publish` read.
_Avoid_: source dir, local package, checkout

**Project**:
The package the user is working on, together with the dependencies it declares.
Only a project has a lock.
_Avoid_: local package, workspace, root package

**Installed Package**:
One version of a package materialized in the package directory, where the Typst
compiler finds it.
_Avoid_: installed copy, cached package

**Editable Install**:
An installed package that is a link to a working tree, so edits to the source
are picked up without reinstalling. The link target is a working tree the user
controls — never a repository clone, which is why `install --remote --editable`
is rejected. A development aid, installed under `@local`; never what a declared
dependency resolves to.
_Avoid_: linked package, dev install, override

**Package Directory**:
The shared directory the Typst compiler resolves packages from, laid out by
namespace, name and version. Machine-global, and gotpm is one writer among
several — the user and other tools install into it too.
_Avoid_: store, package store

**Install Dir**:
A directory named on the command line to receive one package's files directly,
without the namespace, name and version layout around them. Typst cannot import
from it and gotpm does not scan it; it is an output destination, not a package
directory. Only `install`'s working-tree form (`install <path>`) and
`uninstall` take one, because it holds a single package: every command that
installs or deletes a whole dependency graph — `add`, `sync`, `remove`, and
`install --remote`, which performs a Fetch — always works on the package
directory.
_Avoid_: package directory, store

**Namespace**:
The first segment of a package ref. It scopes where the Typst compiler resolves
an import from, and nothing more — `@preview` is the Typst Universe, `@local`
is Typst's own convention for hand-installed packages, `@gotpm` is what a
dependency of a project is imported under. It says nothing about who owns an
installed package.

**Provenance**:
The record kept inside an installed package of the repository and commit it was
built from. It is the only thing that decides whether an installed package is
gotpm's to replace or remove.
_Avoid_: origin, metadata, source info

### Sources

**Repository**:
The git repository a package is developed and released in. It is what its
author pushes to and receives pull requests against, and it is the only kind of
source a dependency may be added from, always pinned to an exact commit.
_Avoid_: remote, source, origin

**Local**:
On this machine. Reserved for that meaning alone — the package directory is
local, a clone is local. Never used for "the package I am developing"; that is
the project.

**Cache**:
State gotpm keeps only to avoid repeating work: the repositories it has cloned
and the Universe index it has fetched. Deleting it loses nothing. The package
directory is never cache.

### Dependencies

**Declared Dependency**:
A package the project's manifest declares and its lock pins. gotpm Fetches it.
Always imported under `@gotpm`.
_Avoid_: dependency (unqualified), requirement

**Imported Package**:
A package a Typst source file imports. Mostly a Typst Universe package, which
the Typst compiler fetches by itself — gotpm only reads such imports and
rewrites their versions.
_Avoid_: dependency, import

Note: the two are different lists and are never assumed to agree. A file may
import a package the project never declared, and a declared dependency no file
imports is still installed. They meet only in the package directory, where a
declared dependency has to be present for an import of it to resolve.

**Lock**:
The record beside a project's manifest naming, for every package it depends on
directly or transitively, the repository and the exact commit it came from. It
is committed, and it is public: anyone who depends on this package reads it to
resolve that package's own dependencies.
_Avoid_: lockfile (as a concept), manifest, dependency list

**Pin**:
One entry of a lock — a package ref bound to a repository and a commit.
_Avoid_: entry, dependency record

**Prune**:
Dropping the pins a lock no longer reaches from any declared dependency, so
removing one dependency also clears the transitive ones only it pulled in. It
happens on its own and never deletes files.
_Note_: `gotpm remove --prune` means something else — it uninstalls the removed
packages from the package directory. The two are unrelated and the flag is
known to be misnamed.

**Drift**:
The condition of a pin whose revision no longer points at the commit that was
pinned. The pin still decides what gets installed; drift is reported, never
acted on.
_Avoid_: stale pin, outdated dependency

### Publication

**Fork**:
A fork of the Typst Universe package repository that submissions are pushed to.
It belongs to the user or to an organisation they publish on behalf of; gotpm
never creates one.
_Avoid_: upstream, universe repo

**Fork Clone**:
The staging area gotpm assembles submissions in: a local clone of one fork. It
holds no work that does not exist elsewhere, so it may be deleted at any time.
A user who publishes through several forks has one clone per fork.
_Avoid_: working repository, fork repo

**Submission**:
One package version staged on a branch of the fork clone, ready to become a
pull request against the Typst Universe. As far as gotpm gets; publication is
the maintainers merging it.
_Avoid_: release, publication, PR

### Operations

**Project Command**:
A command whose subject is the current project: it reads `typst.toml` and
`gotpm.lock`, and installs or deletes the whole dependency graph they describe.
`add`, `sync` and `remove`. Because a graph is many packages, a project command
always works on the package directory and never takes an install dir. `bump` and
`locate` read the project too, but change no dependency.
_Avoid_: dependency command, local command

**Standalone Command**:
A command that needs no project, so the directory it operates on can be named on
the command line. `install`, `uninstall`, `list`, `check`, `publish`, `init`,
`update`, `cache`, `config`, `self`. These are the only commands an install dir
is offered to, and only for the forms of them that act on one package version —
`install`'s working-tree form, not `install --remote`.
_Avoid_: regular command, global command, non-project command

**Install**:
Take a working tree that constitutes a package and place it in the package
directory, so the Typst compiler can find it.

**Fetch**:
Obtain a package version from its repository at an exact commit, together with
everything it depends on, and place it in the package directory.
_Avoid_: download, clone

**Add**:
Fetch a repository as a dependency of the project — confirm it exists, write
its metadata to the manifest and the lock.

**Sync**:
Fetch every dependency the project's lock pins, one after another.

**Remove**:
Drop a declared dependency from the project, and with it the pins nothing else
reaches. Deleting the packages themselves is a separate, opt-in step, because
the package directory is shared.

**Uninstall**:
Delete an installed package from the package directory — one version, every
version of a package, or a whole namespace. Knows nothing about projects.

**Update**:
Rewrite the Typst Universe imports of a source file to the latest published
version of each package. It works on imported packages, not on declared
dependencies, and touches no lock.

**Check**:
Report whether every package a Typst file imports will resolve when it is
compiled. A package outside the Typst Universe has to be present in the package
directory; a Universe package need only exist in the index, because the
compiler downloads it and gotpm does not interfere.

**Publish**:
Get the project's working tree into a fork of the Typst Universe package
repository, ready for a pull request.
