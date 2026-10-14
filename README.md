# GoTPM

[![Go Test](https://github.com/npikall/gotpm/actions/workflows/test.yml/badge.svg)](https://github.com/npikall/gotpm/actions/workflows/test.yml)
[![golangci-lint](https://github.com/npikall/gotpm/actions/workflows/lint.yml/badge.svg)](https://github.com/npikall/gotpm/actions/workflows/lint.yml)

A minimal Typst Package Manager written in Go.

---

This tool is for developers working on Packages for [Typst]. It will make
testing your Package easy, as it allows you to install the package locally and
use it.

- Install Packages into `{data-dir}/typst/packages/`, from a local directory
  or a git repository — including a package in a subdirectory of a monorepo
- Uninstall Packages
- List Packages
- Declare git repositories as dependencies of a project, pinned in a committed
  `gotpm.lock`, so `gotpm sync` reproduces them anywhere
- Install font families from Google Fonts and pin the ones a project needs
- Check that every package a Typst file imports will resolve
- Manage the Version of a Package
- Use it in `GitHub Actions` for publishing Packages to the `Typst Universe`

---

<img class="shadow" src="https://github.com/npikall/gotpm/raw/master/docs/assets/casette.gif">

## Installation

<!-- --8<-- [start:installation] -->

### Unix

**Homebrew:**

```bash
# Short form
brew install npikall/tap/gotpm

# Long form
brew tap npikall/tap && brew install gotpm
```

**Shell:**

```bash
curl -sSfL https://github.com/npikall/gotpm/releases/latest/download/install.sh | sh
```

### Windows

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://github.com/npikall/gotpm/releases/latest/download/install.ps1 | iex"
```

### Install with Go

```bash
go install github.com/npikall/gotpm@latest
```

### Download Binary

Download the Binary from
[GitHub Releases](https://github.com/npikall/gotpm/releases/latest) and place it
in your `$PATH`

### Build from Source

```bash
git clone https://github.com/npikall/gotpm.git
cd gotpm
task install # or read the Taskfile.yml to do build and install manually
```

<!-- --8<-- [end:installation] -->

## Alternatives

<!-- --8<-- [start:alternatives] -->

The main alternative Typst package manager is
[utpm](https://github.com/typst-community/utpm), which targets the same
workflow: developing and publishing local Typst packages.
[typship](https://crates.io/crates/typship) is included for completeness — its
GitHub repository has been deleted and it hasn't been updated since May 2025,
but the crate is still live on crates.io. Feature comparison, accurate as of
October 2026:

| Feature                                | gotpm                                     | utpm `v0.3.0`                    | typship `v0.4.2`             |
| -------------------------------------- | ----------------------------------------- | -------------------------------- | -------------------------------- |
| Language                               | Go                                        | Rust                             | Rust                             |
| Scaffold a package                     | ✅ `gotpm init`                           | ✅ `utpm prj init`               | ✅ `typship init`                |
| Local install                          | ✅ `gotpm install`                        | ✅ `utpm prj link`               | ✅ `typship install <namespace>` |
| Remote install                   | ✅ `gotpm install -r`                     | ✅ `utpm pkg install`            | ✅ `typship download <repo>`     |
| Editable install (symlink)             | ✅ `gotpm install -e`                     | ✅ `utpm prj link --no-copy`     | ✅ `typship dev`                 |
| Publish to Typst Universe              | ✅ `gotpm publish`                        | 🚧 planned  | ✅ `typship publish`             |
| Uninstall                              | ✅ `gotpm uninstall`                      | ✅ `utpm pkg unlink`             | ❌                               |
| List packages                          | ✅ `gotpm list`                           | ✅ `utpm pkg list`               | ❌                               |
| Version bump                           | ✅ `gotpm bump`                           | ✅ `utpm prj bump`               | ❌                               |
| Update deps to latest version          | ✅ `gotpm update`                     | ✅ `utpm prj sync`          | ❌                               |
| Remote install from a subdirectory     | ✅ `gotpm install -r <repo>//<path>` | ❌                               | ❌                               |
| Dependencies pinned in a lock file | ✅ `gotpm add` / `sync` / `remove`   | ❌                               | ❌                               |
| Fonts (install, pin per project)       | ✅ `gotpm font`                      | ❌                               | ❌                               |
| Check that imports resolve             | ✅ `gotpm check`                          | ❌                               | ❌                          |
| Self-update                            | ✅ `gotpm self update`                    | ❌                               | ❌                               |
| Running Pre-, Post-Publish Hooks| ✅ defined in `typst.toml`                     | ❌                               | ❌                               |
| Runtime Dependencies| `git` — publish only                      | `git` — remote install & publish | `git` — remote install only      |

gotpm uses flat top-level commands (`gotpm install`, `gotpm uninstall`) that are
easy to remember. utpm groups commands under `prj`/`pkg` subcommands (e.g.
`utpm prj bump`, `utpm pkg unlink`), which might add some mental overhead in
recalling which group a given command lives under.

<!-- --8<-- [end:alternatives] -->

## Benchmarks

<!-- --8<-- [start:benchmarks] -->

[`scripts/bench.sh`](https://github.com/npikall/gotpm/blob/master/scripts/bench.sh)
uses [hyperfine] to compare gotpm and utpm on install, uninstall, list, bump,
and update/sync. Requires `hyperfine`, `jq`, `git`, `gotpm`, and `utpm` on
`$PATH`. Everything runs against sandboxed temp directories — it never touches
your real Typst package directory.

```bash
./scripts/bench.sh
```

<!-- BENCH:RESULTS:START -->

| Command              | gotpm  | utpm   | Result             |
| -------------------- | ------ | ------ | ------------------ |
| Install (local)      | 4 ms   | 5 ms   | gotpm 1.1x faster  |
| Install (remote git) | 994 ms | 876 ms | utpm 1.1x faster   |
| Uninstall            | 5 ms   | 4 ms   | utpm 1.0x faster   |
| List                 | 5 ms   | 5 ms   | gotpm 1.1x faster  |
| Bump (patch)         | 5 ms   | 8 ms   | gotpm 1.7x faster  |
| Update (18 imports)  | 230 ms | 5.92 s | gotpm 25.8x faster |

*Benchmarked 2026-10-14 on Darwin 25.6.0 arm64 with gotpm

v0.7.0 and utpm 0.3.0. Results vary by machine and network conditions - run
`scripts/bench.sh` yourself to reproduce. Whichever tool is faster for a given
command is reported as-is.*
<!-- BENCH:RESULTS:END -->

[hyperfine]: https://github.com/sharkdp/hyperfine

<!-- --8<-- [end:benchmarks] -->

![Gopher](https://raw.githubusercontent.com/egonelbre/gophers/master/.thumb/vector/projects/network-side.png)

[Typst]: https://typst.app
