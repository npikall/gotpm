---
icon: lucide/user
---

# Useage

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

Uninstall a package.

```console
$ gotpm help uninstall
Uninstall a Typst Package from the local Storage

USAGE
    gotpm uninstall [name] [--flags]

EXAMPLES
    gotpm uninstall # get name and version from typst.toml
    gotpm uninstall foo
    gotpm uninstall foo --namespace preview
    gotpm uninstall foo --namespace preview --dry-run

FLAGS
    --all           Uninstall all Packages from a given namespace or all versions of a package.
    --dry-run       Perform a dry run.
    -h --help       Help for uninstall
    -n --namespace  The namespace from which the package should be removed from. (local)
    -V --verbose    Print Debug Level Information
    -v --version    The specific version of a package that should be removed.
```
