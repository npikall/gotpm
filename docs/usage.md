---
icon: lucide/user
---

# Usage

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
$ gotpm help init
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

The `packages` path follows the same overrides the install commands do —
`$GOTPM_INSTALL_DIR` first, then `$TYPST_PACKAGE_PATH` — and the listing notes
which one applied:

```console
$ GOTPM_INSTALL_DIR=/tmp/scratch gotpm locate
Typst
  packages   /tmp/scratch (via $GOTPM_INSTALL_DIR)
...
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
