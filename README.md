# GoTPM

[![Go Test](https://github.com/npikall/gotpm/actions/workflows/test.yml/badge.svg)](https://github.com/npikall/gotpm/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/npikall/gotpm)](https://goreportcard.com/report/github.com/npikall/gotpm)

A minimal Typst Package Manager written in Go.

---

This tool is for developers working on Packages for [Typst]. It will make
testing your Package easy, as it allows you to install the package locally
and use it.

- Install Packages into `{data-dir}/typst/packages/`
- Uninstall Packages
- List Packages
- Manage the Version of a Package
- Use it in `GitHub Actions` for publishing Packages to the `Typst Universe`

---

<img class="shadow" src="https://github.com/npikall/gotpm/raw/main/docs/assets/casette.gif">

## Installation

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

Download the Binary from [GitHub Releases](https://github.com/npikall/gotpm/releases/latest) and place it in your `$PATH`

### Build from Source

```bash
git clone https://github.com/npikall/gotpm.git
cd gotpm
just install # or read the justfile to do build and install manually
```

## Alternatives

The main alternative Typst package manager is [utpm](https://github.com/typst-community/utpm),
which targets the same workflow: developing and publishing local Typst packages.
[typship](https://crates.io/crates/typship) is included for completeness — its
GitHub repository has been deleted and it hasn't been updated since May 2025,
but the crate is still live on crates.io. Feature comparison, accurate as of
August 2026:

| Feature                       | gotpm                               | utpm `v0.3.0`                            | typship `v0.4.2` [^4]                |
| ----------------------------- | ----------------------------------- | ----------------------------------------- | -------------------------------------- |
| Language                      | Go                                 | Rust                                     | Rust                                  |
| Local install                 | ✅ `gotpm install`                  | ✅ `utpm prj link`                         | ✅ `typship install <namespace>`        |
| Remote install (git)          | ✅ `gotpm install -r`               | ✅ `utpm pkg install`                      | ✅ `typship download <repo>`            |
| Editable install (symlink)    | ✅ `gotpm install -e`               | ✅ `utpm prj link --no-copy`               | ❌                                      |
| Uninstall                     | ✅ `gotpm uninstall`                | ✅ `utpm pkg unlink`                       | ❌                                      |
| List packages                 | ✅ `gotpm list`                     | ✅ `utpm pkg list`                         | ❌                                      |
| Version bump                  | ✅ `gotpm bump`                     | ✅ `utpm prj bump`                         | ❌                                      |
| Publish to Typst Universe     | ✅ `gotpm publish`                  | 🚧 planned (`utpm prj publish`)            | ✅ `typship publish`                    |
| Update deps to latest version | ✅ `gotpm update` [^1]              | ✅ `utpm prj sync` [^2]                    | ❌                                      |
| External binaries required [^3] | `git` — publish only          | `git` — remote install & publish     | `git` — remote install only            |

[^1]: Fetches the Typst Universe version index once, then resolves all
    import statements in that file concurrently (goroutines) against the
    in-memory index — one network request regardless of import count.

[^2]: Despite `async fn` signatures, [`sync.rs`](https://github.com/typst-community/utpm/blob/ccdf834320f56df4aa1277dc931a3d3fd72c4af1/src/commands/sync.rs)
    files and imports seem to be processed one at a time in plain `for` loops. Each
    `@preview` import seems to trigger a fresh, uncached HTTP request to the Typst
    Universe registry, so N imports means N sequential round-trips.

[^3]: Only `gotpm publish` shells out to a system `git` binary via
    [`internal/gitcli`](https://github.com/npikall/gotpm/blob/main/internal/gitcli/gitcli.go),
    needed for sparse checkout and commit signing that the pure-Go `go-git`
    library can't do. Every other command, including `gotpm install -r`,
    uses `go-git` and needs no external binary. utpm's `pkg install`
    (doc-commented "requires git to be installed") and `prj publish` both
    shell out to a system `git` binary via
    [`utils/git.rs`](https://github.com/typst-community/utpm/blob/main/src/utils/git.rs)
    (`std::process::Command`) for clone/add/commit/push/pull. typship's
    `download` command also shells out to `git clone`/`git checkout`
    (`src/commands/download.rs`); `typship publish` uses the GitHub API
    (`octocrab`) directly and needs no git binary.

[^4]: Last published 2025-05-07 (`v0.4.2`). Its GitHub repository
    (`sjfhsjfh/typship`, linked from the crates.io metadata) now 404s, so the
    source used here comes from the crate tarball on crates.io
    (`typship-0.4.2.crate`) rather than the repo.

gotpm uses flat top-level commands (`gotpm install`, `gotpm uninstall`) that are
easy to remember. utpm groups commands under `prj`/`pkg` subcommands (e.g.
`utpm prj bump`, `utpm pkg unlink`), which might add some mental overhead in recalling
which group a given command lives under.

![Gopher](https://raw.githubusercontent.com/egonelbre/gophers/master/.thumb/vector/projects/network-side.png)

[Typst]: https://typst.app
