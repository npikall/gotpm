# The manifest and the config are edited in place

`typst.toml` is written and commented by hand and stays the author's, and
`config.toml` is the user's own file. gotpm used to re-encode the whole
manifest on `bump` and the whole config on `config set` — dropping comments and
reordering keys — and kept a hand-written line editor just for the
`[tool.gotpm]` arrays. Every write to a TOML file now goes through
`github.com/npikall/toml-edit`, which changes only the bytes of the value being
set and leaves everything else as it was. Reading still decodes into structs
with `github.com/BurntSushi/toml`, because tomledit has no struct decoding; a
later change may move decoding over too.

## Considered Options

- **Keep re-encoding with BurntSushi.** Rejected: it discards the author's
  comments and key order on every `bump`, contradicting that the manifest stays
  the author's.
- **Keep the hand-written line editor and extend it to `bump` and the config.**
  Rejected: it only understood `[tool.gotpm]` written as a section, and growing
  it is growing a second TOML parser inside gotpm.
- **Add struct decoding to tomledit and drop BurntSushi now.** Deferred:
  tomledit's job is format-preserving editing, and a decoder is a second public
  API to commit to. Two parsers per file is the price meanwhile.

## Consequences

- Every TOML file gotpm writes is parsed twice — BurntSushi to read it, tomledit
  to edit it. A file one accepts and the other rejects (TOML 1.1 syntax, say)
  fails as an invalid manifest or config rather than being rewritten.
- `bump` sets only the keys whose value changed, so a version bump is a one-line
  diff; `--indent` controls nothing any more and is kept hidden as a no-op.
- `[tool.gotpm]` may be written inline or with dotted keys; gotpm no longer
  rejects those forms. An array that was written across several lines stays
  one element per line, and an emptied `[tool.gotpm]` is removed — `[tool]`
  is left alone.
