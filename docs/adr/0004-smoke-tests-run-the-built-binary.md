# Smoke tests run the built binary

Every command already has a thorough test beside it under `internal/cmds`, but
those tests call `Run` directly and start below the cobra layer, so nothing
covered flag wiring, command registration, argument validation or the exit code
a caller sees. The smoke suite in `internal/smoke` builds the binary once in
`TestMain` and runs each command as a subprocess, asserting the exit code and
one cheap outcome. It is deliberately shallow: the behaviour of a command is
tested where the command lives, and the smoke suite exists only to prove that
the path from `main` to that behaviour is intact.

## Considered Options

- **Drive `rootCmd` in-process.** Rejected: it covers the same wiring far more
  cheaply, but `rootCmd` is a package-level singleton whose flags keep their
  values between executions, so a suite of them either leaks state or forces a
  `newRootCmd` refactor — and it still never runs `main`, never produces an exit
  code and never proves the binary links.
- **Assert on stderr as well as the exit code.** Rejected: the spinner and
  `internal/ui` own that stream, and coupling the suite to their rendering buys
  nothing the exit code and an outcome check do not already give.

## Consequences

- The subprocess is given an explicit environment, never an inherited one. This
  is what keeps `GITHUB_TOKEN`, `GH_TOKEN` and `SSH_AUTH_SOCK` out of a test:
  with an inherited environment a developer or runner holding real push
  credentials would hand them to a `publish` test, and a fixture pointing
  somewhere other than its `file://` fork could authenticate against a real
  remote. An allowlist makes that impossible rather than unlikely.
- Isolation has to be expressible as data, not only as `t.Setenv` calls, so
  three helpers grew a path- or env-explicit core with the existing convenience
  wrapper on top: `testrepo.Env` under `Isolate`, `index.SaveCacheAt` under
  `SaveCache`, and `testrepo.ProjectAt` under `Project`. Test code no longer
  re-derives paths the production code owns.
- Closing the environment allowlist surfaced a gap in `Isolate` itself: it never
  set `XDG_CONFIG_HOME`, which `config.Path` reaches through
  `os.UserConfigDir`, so the existing config and publish tests read the
  developer's real gotpm configuration whenever that variable was exported.
- Isolation now means a controlled machine rather than a broken one.
  `GIT_CONFIG_GLOBAL` points at a file inside the isolated root instead of
  `os.DevNull`, because publishing commits as the user and correctly refuses to
  run where git has no identity — a fork clone gotpm creates itself cannot be
  configured by the test beforehand. Nothing creates that file, and git reads a
  missing config as an empty one, so a test gets an identity only by asking for
  it. Fixture repositories carry the same idea in their own config: they are
  created on `main` and with `commit.gpgsign` off, so they build the same way
  for a developer who signs their commits and one who does not.
- Every command is reachable offline, and each for its own reason: `check` and
  `update` read a seeded index cache inside the isolated data directory,
  `publish --local` commits to a `file://` fork and never pushes, and `add` and
  `sync` clone `file://` repositories. `self update` is the exception and is not
  covered — on a `dev` build it refuses before reaching the network, which is a
  refusal, not a command exercised.
- The suite runs under `go test ./...` and skips under `-short`, so it cannot
  drift out of compilation the way a build-tagged suite does. It runs on Linux
  only. Running it on the platforms the release workflow ships to is where a
  subprocess suite pays for itself, and it is blocked on `testrepo` building
  repository URLs as `"file://" + dir`, which is malformed on Windows.
