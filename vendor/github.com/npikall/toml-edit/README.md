# tomledit

Format-preserving TOML for Go. Read a TOML file, change a few values, write
it back: comments, blank lines, indentation, key quoting and the spelling of
untouched values stay byte for byte the same.

- **TOML 1.1** (multi-line inline tables, `\xHH` and `\e` escapes, optional
  seconds), with an opt-in **TOML 1.0** strict mode.
- **Lossless:** `Parse(src).String() == src` for every valid document.
- **Conformant:** passes all valid and invalid cases of
  [toml-test](https://github.com/toml-lang/toml-test) for 1.0 and 1.1.
- **No runtime dependencies.**

```sh
go get github.com/npikall/toml-edit
```

## Usage

```go
doc, err := tomledit.Parse(src)
if err != nil {
	return err
}

port, err := doc.GetInt("server", "port")

err = doc.Set([]string{"server", "port"}, 9090)                          // replace a value
err = doc.Insert([]string{"server", "workers"}, 4)                       // add a key
err = doc.Insert([]string{"server", "limits"}, map[string]any{"cpu": 2}) // inline table
err = doc.Delete("server", "debug")                                      // remove a key or table

os.WriteFile("config.toml", []byte(doc.String()), 0o644)
```

Given

```toml
# server settings
[server]
host = "localhost"  # bind address
port = 8080
debug = true
```

`Set`, `Insert` and `Delete` above produce a minimal diff:

```diff
 # server settings
 [server]
 host = "localhost"  # bind address
-port = 8080
-debug = true
+port = 9090
+workers = 4
+limits = { cpu = 2 }
```

### Editing rules

| Operation | Behavior |
|---|---|
| `Set` | Replaces only the value text. Key, spacing and trailing comment stay. The path must exist and hold a value, not a table. An array written across several lines stays one element per line, with its indentation and a trailing comma; elements that remain keep their spelling and comments. |
| `Insert` | Appends to the end of the table section that owns the key, indented like its last key. Keys under a dotted-key table stay dotted, keys of inline tables go into the braces, keys of missing tables get a new `[header]` at the end of the file. With `Multiline()`, an array is written one element per line. |
| `Delete` | Removes the key with the comments and blank lines before it. Deleting a table removes its header, body and sub-tables. Commas in inline tables are fixed up. |

Every edit is re-validated. An edit that would make the document invalid (for
example a conflicting key) is rolled back and returns an error. Errors wrap
`ErrNotFound`, `ErrNotValue`, `ErrExists`, `ErrNotTable` or `ErrType` for use
with `errors.Is`.

### Values

| TOML | Go (read) | Go (write) |
|---|---|---|
| string | `string` | `string` |
| integer | `int64` | any integer type (up to `math.MaxInt64`) |
| float | `float64` | `float32`, `float64` |
| boolean | `bool` | `bool` |
| offset date-time | `time.Time` | `time.Time` |
| local date-time / date / time | `eval.LocalDateTime` / `eval.LocalDate` / `eval.LocalTime` | same |
| array | `[]any` | slices and arrays |
| table | `*eval.Table` | `map[string]V` (written as inline table) |

New text is always valid TOML 1.0 and 1.1.

### TOML 1.0

```go
doc, err := tomledit.Parse(src, tomledit.Strict10())
```

rejects syntax added in 1.1. Edits never introduce 1.1 syntax, so a strict
document stays valid 1.0.

## Packages

| Package | Role |
|---|---|
| `tomledit` | Public edit API |
| `token`, `lexer` | Tokens and context-sensitive lexer |
| `ast` | Lossless concrete syntax tree |
| `parser` | Recursive-descent parser (syntax only) |
| `eval` | Semantic validation and value decoding |
| `format` | Go values to canonical TOML text |
| `cmd/toml-test-decoder` | Decoder for the toml-test runner |

## Development

Requires Go and [Task](https://taskfile.dev).

```sh
task test        # unit, golden and toml-test conformance tests
task lint        # golangci-lint
task bench       # benchmarks
task toml-test   # official toml-test runner, TOML 1.0 and 1.1
task fuzz        # lexer; fuzz:parser and fuzz:eval for the others
```
