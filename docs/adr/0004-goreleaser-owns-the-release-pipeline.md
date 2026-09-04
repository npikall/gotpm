# GoReleaser owns the release pipeline

The build-and-publish half of releasing gotpm — cross-compiling binaries,
checksuming them, writing the GitHub release, pushing the Homebrew formula —
was a hand-rolled OS/arch matrix plus a few shell steps in
`.github/workflows/release.yml`. We replaced that half with GoReleaser (OSS,
no paid features). `task release:next` (git-cliff changelog + commit + `git
tag`) and the manual `git push --tags` that triggers the workflow are
unchanged — GoReleaser only owns what happens once CI picks up the tag.

## Considered Options

- **Asset shape**: raw binaries via `archives: formats: [binary]`, using
  GoReleaser's own `{name}_{version}_{os}_{arch}` naming — not `.tar.gz`/`.zip`
  archives. Every current consumer (curl\|sh, the PowerShell installer,
  `self update`'s asset filter, the "download the binary" README channel)
  expects a runnable file with no unpack step, and archives buy nothing they
  don't already get some other way. Revisit if `mise` support is ever added
  (tracked in `TODO.md`) — generic installer tools like that are tuned around
  the archive convention.
- **CI topology**: one `ubuntu-latest` job cross-compiling all six targets,
  replacing the per-OS matrix and the separate build/release jobs. We already
  cross-compiled Windows from Linux; `CGO_ENABLED=0` throughout means nothing
  in the build is host-OS-dependent, so the `macos-latest` runner wasn't
  buying us anything either.
- **Homebrew**: GoReleaser's `homebrew_casks:` pipe now generates
  `Casks/gotpm.rb` and pushes it to `npikall/homebrew-tap` on every release,
  pointing at a prebuilt binary — replacing the hand-maintained,
  build-from-source Formula and `scripts/update-formulas.sh`'s gotpm entry.
  (The classic `brews:` pipe, which produces a Formula, is deprecated in this
  GoReleaser version in favor of `homebrew_casks:` — confirmed against the
  installed binary, not just docs, after `brews:` and then even
  `homebrew_casks.binary` each turned out to be flagged deprecated in turn.
  User-facing install commands are unaffected; Homebrew resolves `brew
  install <name>` to a cask the same way it does a formula.) As a side effect
  this fixes an existing drift bug: the hand-written formula's ldflags never
  stamped `gitCommit` the way the Taskfile build always did. One build
  definition, no more drift.
- **Detecting a Homebrew install**: with one shared build, there's no longer
  a brew-specific compile step to stamp `installer=brew` into. `self update`
  instead checks at runtime whether its own executable path sits under a
  known Homebrew Cellar prefix. The alternative — a second GoReleaser build
  target stamped `installer=brew`, wired only into the `brews:` formula —
  would've doubled the binaries published on every release for artifacts
  nobody downloads directly.
- **Signing**: keyless cosign via GitHub Actions OIDC, signing `checksums.txt`
  once — not a static keypair (no secret to generate, store, rotate, or leak)
  and not per-artifact signatures (GoReleaser's own documented recommendation:
  once every binary's hash is in the checksum file, signing that file once is
  sufficient).
- **Release notes**: unchanged. git-cliff still renders them from `cliff.toml`;
  GoReleaser consumes the rendered file via `--release-notes` instead of using
  its own changelog pipe, which doesn't reproduce the custom grouping/footer
  templates already in place.
- **Draft releases**: kept (`release: draft: true`), preserving the existing
  manual-review-before-publish step. `--fail-on-no-commits` (a guard against
  re-tagging with nothing new to report) is dropped without a replacement —
  a minor edge case, and an empty-looking draft would be obvious before
  publishing it anyway.

## Consequences

- `internal/cmds/self/self.go`'s release-asset filter and both
  `scripts/install.*.tmpl` templates need rewriting for the new,
  version-embedded, underscore-separated naming convention.
- `npikall/homebrew-tap` needs manual, out-of-band changes (a new cross-repo
  PAT secret on the `gotpm` side included) — see `HOMEBREW-TAP-MIGRATION.md`.
- README gains a "Verify" section documenting `cosign verify-blob` against
  the signed `checksums.txt`.
- A new `release-test` Task target (`goreleaser release --snapshot --clean
  --skip=sign`) gives a local dry-run of the build/package/checksum steps.
  `--skip=sign` is required, not optional: this GoReleaser version's `signs`
  config has no per-step conditional (no `if:` field), and `--snapshot`
  alone does not skip signing — an unguarded local run would either hang
  waiting for cosign's interactive OIDC browser flow or fail outright, since
  no CI OIDC identity exists on a local machine.
- Detecting a Homebrew install is therefore two checks, both in
  `internal/cmds/self`. `BuildInfo.Installer == "brew"` covers binaries built
  by the pre-GoReleaser formula, which stamped that ldflag; the path check
  covers every install since. It looks for the `Cellar` segment, which the
  macOS Intel (`/usr/local`), Apple Silicon (`/opt/homebrew`) and Linuxbrew
  (`/home/linuxbrew/.linuxbrew`) prefixes all share, and resolves symlinks
  first, because such a binary is normally reached through `<prefix>/bin`
  rather than by its Cellar path.
- The asset filter matches on the `_<os>_<arch>` suffix rather than the
  version-bearing prefix of `gotpm_<tag>_<os>_<arch>[.exe]`: the version being
  looked up is not known in advance.
