---
title: Installation
icon: lucide/download
---

# Installation

gotpm ships as a single static binary. There is no runtime to install and no Go
toolchain needed unless you build it yourself.

--8<-- "README.md:installation"

## Verifying a release

Every release's `checksums.txt` is signed keylessly with
[cosign](https://docs.sigstore.dev/cosign/system_config/installation/), tying it
to the exact `release.yml` GitHub Actions run that built it. Checking a
downloaded binary takes two steps: verify the signature on `checksums.txt`, then
verify the binary against the checksum it lists.

!!! note "cosign v2 or newer" `--certificate-identity-regexp` and
`--certificate-oidc-issuer` do not exist in cosign v1, which fails with
`unknown flag`. Check with `cosign version`.

Pick the release and the binary for your platform, then download the binary, the
checksums and their signature bundle into one directory:

```bash
VERSION=v0.7.0
BINARY=gotpm_${VERSION}_darwin_arm64
BASE=https://github.com/npikall/gotpm/releases/download/$VERSION
curl -sSfLO "$BASE/$BINARY"
curl -sSfLO "$BASE/checksums.txt"
curl -sSfLO "$BASE/checksums.txt.sigstore.json"
```

Binaries are named `gotpm_<version>_<os>_<arch>`, with `linux`, `darwin` or
`windows` and `amd64` or `arm64`; the Windows ones end in `.exe`.

Verify the signature. cosign prints `Verified OK` when `checksums.txt` was
signed by gotpm's release workflow for a tagged release:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/npikall/gotpm/\.github/workflows/release\.yml@refs/tags/v.*$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

Then check the binary against the now-trusted checksum:

```bash
sha256sum -c checksums.txt --ignore-missing
```

On a macOS without `sha256sum`,
`shasum -a 256 -c checksums.txt --ignore-missing` does the same. Either prints
`<binary>: OK`.

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

The one that matters most is `packages`: the
[package directory](concepts.md#the-package-directory) the Typst compiler
resolves imports from. It is shared with Typst itself and with anything else
that installs there.

## Shell completion

Cobra's completion command is available for the common shells:

```console
$ gotpm completion zsh > "${fpath[1]}/_gotpm"
$ gotpm completion bash > /etc/bash_completion.d/gotpm
$ gotpm completion fish > ~/.config/fish/completions/gotpm.fish
```
