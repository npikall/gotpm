---
title: Installation
icon: lucide/download
---

# Installation

gotpm ships as a single static binary. There is no runtime to install and no
Go toolchain needed unless you build it yourself.

--8<-- "README.md:installation"

## Verifying the install

```console
$ gotpm self version
```

`gotpm self update` fetches the latest release from GitHub and replaces the
running binary in place — useful when you installed from the shell script or a
release download. Package-manager installs are better updated through the
package manager that placed them.

## Where gotpm puts things

Nothing is written until a command needs it. `gotpm locate` prints every path
gotpm reads or writes, including the ones that do not exist yet:

```console
$ gotpm locate
```

The one that matters most is `packages`: the [package
directory](concepts.md#the-package-directory) the Typst compiler resolves
imports from. It is shared with Typst itself and with anything else that
installs there.

## Shell completion

Cobra's completion command is available for the common shells:

```console
$ gotpm completion zsh > "${fpath[1]}/_gotpm"
$ gotpm completion bash > /etc/bash_completion.d/gotpm
$ gotpm completion fish > ~/.config/fish/completions/gotpm.fish
```
