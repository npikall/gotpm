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

Locate the Root Path, where all Typst Packages get installed into.

```console
$ gotpm help locate
Locate the root directory, where the Typst Packages are stored.

USAGE
    gotpm locate [--flags]

EXAMPLES
    gotpm locate

FLAGS
    -h --help  Help for locate
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
