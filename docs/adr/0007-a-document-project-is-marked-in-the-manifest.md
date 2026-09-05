# A document project is marked in the manifest

`gotpm init` scaffolds two different things. A package is source other projects
import and the compiler never renders on its own; a document project is source
compiled into a PDF or an HTML page that nothing ever imports. A document
project still declares dependencies, so it still carries a `typst.toml` — but
that manifest describes something the Typst Universe will never see, and the
files on disk are the only difference between the two, `main.typ` against
`lib.typ`. So a document project is marked explicitly, with `kind = "document"`
under `[tool.gotpm]`, the section gotpm already owns. A manifest without the key
is a package, which is what every manifest written before this field existed is.

## Considered Options

- **Leave `version` and `entrypoint` empty, and let the emptiness be the
  marker.** This is what the domain says: a document project has no version
  because it is never published, and no entrypoint because nothing imports it.
  Rejected because `validateManifest` requires name, version and entrypoint,
  and `deps.OpenProject` runs it before every project command. A manifest with
  both fields blank fails to load, so `gotpm add` — the one command the
  document manifest exists for — exits 1 with `missing required field:
  package.version`. Making validation kind-aware would fix that, but only by
  making blank-versus-filled load-bearing across the whole manifest package,
  and the kind would still have to be inferred rather than stated.
- **Write no marker at all and infer the kind from the entrypoint filename.**
  Rejected: `main.typ` against `lib.typ` is a convention this scaffolding
  happens to follow, not a fact about the project. An author renaming their
  entrypoint would change what their project *is*.
- **Write `kind = "package"` too, so every manifest states its kind.**
  Rejected: absence already means package, so the key would carry no
  information, and it would put a `[tool.gotpm]` section into every scaffolded
  package for the sake of a default.

## Consequences

- `version = "0.1.0"` and `entrypoint = "main.typ"` are written into a document
  project's manifest and are meaningless there. They are the price of the
  manifest format being the package manifest format, and the alternative was
  breaking `gotpm add`. Nothing should start deriving meaning from them.
- Nothing reads the kind yet. `gotpm install` will place a document project in
  the package directory as `@local/thesis:0.1.0` and `gotpm publish` will stage
  it for the Typst Universe. Refusing both is a separate change; the marker is
  what makes it possible without a further manifest change.
- The marker lives in the section `SetDependencies` rewrites. That function is
  line-based and splices only the dependency array, and it keeps a
  `[tool.gotpm]` section that still holds another key when the last dependency
  goes — so `gotpm remove` does not silently demote a document project to a
  package. That behaviour is pinned by a test, because it falls out of how
  `removeArray` decides a section is empty rather than from anything explicit.
- A future kind — a Typst Universe *template*, which is a real thing the
  manifest format already reserves a `[template]` section for — is a third
  value of this key rather than a second special case.
