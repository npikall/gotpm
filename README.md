# GoTPM

[![Go Test](https://github.com/npikall/gotpm/actions/workflows/ci.yml/badge.svg)](https://github.com/npikall/gotpm/actions/workflows/test.yml)

A minimal Typst Package Manager written in Go.

---

This tool is for developers working on Packages for [Typst]. It will make
testing your Package, as it allows you to easily install the package locally
and use it.

- Install Packages into `{data-dir}/typst/packages/`
- Uninstall Packages
- List Packages
- Manage the Version of a Package
- Use it in `GitHub Actions` for publishing Packages to the `Typst Universe`

---

```console
Usage:
  gotpm [command]

Available Commands:
  completion              Generate the autocompletion script for the specified shell
  help                    Help about any command
  install                 Install a Typst Package locally.
  list                    List all locally installed Packages
  uninstall               Uninstall a Typst Package from the local Storage
  version                 Manage the version of a Typst Package


Flags:
  -h, --help     help for gotpm
  -t, --toggle   Help message for toggle

Use "gotpm [command] --help" for more information about a command.
```

# Installation

## Quick Install

```console
curl -sSfL https://github.com/npikall/gotpm/releases/latest/download/install.sh | sh
```

## Install with Go

```console
go install github.com/npikall/gotpm@latest
```

## Download Binary

Download the Binary from [GitHub Releases](https://github.com/npikall/gotpm/releases/latest) and place it in you `$PATH`

## Build from Source

```console
git clone https://github.com/npikall/gotpm.git
cd gotpm
make install # or read the Makefile to do build and install manually
```

![Gopher](https://raw.githubusercontent.com/egonelbre/gophers/master/.thumb/vector/projects/network-side.png)

[Typst]: https://typst.app
