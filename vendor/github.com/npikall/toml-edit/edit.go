// Package tomledit reads and edits TOML documents while preserving their
// formatting: comments, whitespace, key quoting and the spelling of values
// that are not changed survive every edit byte for byte.
//
// Parse accepts TOML 1.1 by default, or only TOML 1.0 with the Strict10
// option. A parsed Document returns its source unchanged from String until
// it is edited with Set, Insert or Delete. Every edit is validated; an edit
// that would make the document invalid is rolled back and reported.
//
// Values are read with Get or the typed getters (GetInt, GetString, ...).
// Integers are int64, floats float64, offset date-times time.Time, and
// local dates and times the types of package eval. New values are written
// from Go strings, booleans, integers, floats, time.Time, the eval
// date/time types, slices (as arrays) and maps with string keys (as inline
// tables).
package tomledit

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/format"
	"github.com/npikall/toml-edit/parser"
)

var (
	// ErrNotFound is returned when a key path does not exist.
	ErrNotFound = errors.New("key not found")
	// ErrNotValue is returned by Set when a key path names a table defined
	// by a header or dotted keys, or an array of tables.
	ErrNotValue = errors.New("key does not hold a value")
	// ErrType is returned by the typed getters when the value has a
	// different type.
	ErrType = errors.New("wrong value type")
	// ErrExists is returned by Insert when the key path already exists.
	ErrExists = errors.New("key already exists")
	// ErrNotTable is returned by Insert when the parent of the key path is
	// not a table.
	ErrNotTable = errors.New("key does not hold a table")
)

// Document is a parsed TOML document. Its String method returns the source
// with all edits applied.
type Document struct {
	cst  *ast.Document
	root *eval.Table
}

// Option configures Parse.
type Option = parser.Option

// Strict10 makes Parse accept TOML 1.0 only, rejecting the syntax that 1.1
// added. Edits never add 1.1 syntax, so an edited document stays valid 1.0.
func Strict10() Option { return parser.Strict10() }

// Parse parses and validates src as TOML 1.1, or 1.0 with Strict10. The
// result's String method returns src unchanged until it is edited.
func Parse(src string, opts ...Option) (*Document, error) {
	cst, err := parser.Parse(src, opts...)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	root, err := eval.Evaluate(cst)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	return &Document{cst: cst, root: root}, nil
}

// String returns the document as TOML text.
func (d *Document) String() string { return d.cst.String() }

// Get returns the value at the key path: a string, int64, float64, bool,
// time.Time, eval.LocalDateTime, eval.LocalDate, eval.LocalTime, []any or
// *eval.Table. No keys return the root table.
func (d *Document) Get(keys ...string) (any, bool) {
	var v any = d.root
	for _, key := range keys {
		t, ok := v.(*eval.Table)
		if !ok {
			return nil, false
		}
		if v, ok = t.Get(key); !ok {
			return nil, false
		}
	}
	return v, true
}

// GetInt returns the integer at the key path.
func (d *Document) GetInt(keys ...string) (int64, error) { return get[int64](d, keys) }

// GetFloat returns the float at the key path.
func (d *Document) GetFloat(keys ...string) (float64, error) { return get[float64](d, keys) }

// GetString returns the string at the key path.
func (d *Document) GetString(keys ...string) (string, error) { return get[string](d, keys) }

// GetBool returns the boolean at the key path.
func (d *Document) GetBool(keys ...string) (bool, error) { return get[bool](d, keys) }

// Set replaces the value of an existing key. Only the value's text changes;
// the key, the whitespace around the value and a trailing comment are kept.
// An array written across several lines stays one element per line with a
// trailing comma; elements that remain keep their spelling and comments.
func (d *Document) Set(path []string, value any) error {
	if err := d.set(path, value); err != nil {
		return fmt.Errorf("%s: %w", dotted(path), err)
	}
	return nil
}

// EditOption configures Insert. Set takes none: it follows the layout of
// the value it replaces.
type EditOption func(*editConfig)

type editConfig struct {
	multiline bool
}

// Multiline makes Insert write an array value one element per line, each
// followed by a comma and indented two spaces deeper than the line of its
// key.
func Multiline() EditOption {
	return func(c *editConfig) { c.multiline = true }
}

// Insert adds a new key with value at the end of the table section that
// owns it, indented like the key/value before it.
func (d *Document) Insert(path []string, value any, opts ...EditOption) error {
	var cfg editConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := d.insert(path, value, cfg); err != nil {
		return fmt.Errorf("%s: %w", dotted(path), err)
	}
	return nil
}

// Delete removes the key at the key path together with the comments and
// blank lines before it. Deleting a table removes its header, its key/values
// and all its sub-tables.
func (d *Document) Delete(keys ...string) error {
	if err := d.delete(keys); err != nil {
		return fmt.Errorf("%s: %w", dotted(keys), err)
	}
	return nil
}

func (d *Document) delete(keys []string) error {
	if _, ok := d.Get(keys...); !ok || len(keys) == 0 {
		return ErrNotFound
	}
	return d.mutate(func() { deletePath(d.cst, keys) })
}

func (d *Document) set(path []string, value any) error {
	kv, err := d.keyValue(path)
	if err != nil {
		return err
	}
	node, err := format.Value(value)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}
	if oldArr, ok := kv.Value.(*ast.Array); ok && multiline(oldArr) {
		if arr, ok := node.(*ast.Array); ok {
			decoded, _ := d.Get(path...) // keyValue found path
			oldValues, _ := decoded.([]any)
			layout := layoutOf(oldArr, oldValues, d.newline())
			layout.apply(arr)
		}
	}
	*ast.ValueDecor(node) = *ast.ValueDecor(kv.Value)
	old := kv.Value
	kv.Value = node
	root, err := eval.Evaluate(d.cst)
	if err != nil {
		kv.Value = old
		return fmt.Errorf("evaluate: %w", err)
	}
	d.root = root
	return nil
}

// keyValue returns the CST node that defines the value at path.
func (d *Document) keyValue(path []string) (*ast.KeyValue, error) {
	if len(path) == 0 {
		return nil, ErrNotFound
	}
	parent, _ := d.Get(path[:len(path)-1]...)
	t, ok := parent.(*eval.Table)
	if !ok {
		return nil, ErrNotFound
	}
	last := path[len(path)-1]
	if _, ok := t.Get(last); !ok {
		return nil, ErrNotFound
	}
	kv, ok := t.KeyValue(last)
	if !ok {
		return nil, ErrNotValue
	}
	return kv, nil
}

func (d *Document) insert(path []string, value any, cfg editConfig) error {
	if len(path) == 0 {
		return ErrNotFound
	}
	if _, ok := d.Get(path...); ok {
		return ErrExists
	}
	node, err := format.Value(value)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}
	// Walk to the deepest existing table on the path, remembering the
	// nearest section that holds key/values for it.
	table, depth := d.root, 0
	owner, ownerDepth := table.Section(), 0
	var inline *ast.InlineTable
	inlineDepth := 0
	for _, key := range path[:len(path)-1] {
		v, ok := table.Get(key)
		if !ok {
			break
		}
		if table, ok = v.(*eval.Table); !ok {
			return ErrNotTable
		}
		depth++
		if section := table.Section(); section != nil {
			owner, ownerDepth = section, depth
		}
		if node := table.Inline(); node != nil {
			inline, inlineDepth = node, depth
		}
	}
	nl := d.newline()
	spreadArray := func(indent string) {
		if arr, ok := node.(*ast.Array); ok && cfg.multiline {
			layout := newLayout(indent, nl)
			layout.apply(arr)
		}
	}
	switch {
	case inline != nil:
		spreadArray(sectionIndent(owner))
		kv := newKeyValue(path[inlineDepth:], node, "")
		return d.mutate(func() { appendEntry(inline, kv) })
	case table.Section() != nil && depth == len(path)-1, table.Dotted():
		spreadArray(sectionIndent(owner))
		kv := newKeyValue(path[ownerDepth:], node, nl)
		return d.mutate(func() { appendKeyValue(owner, kv) })
	default:
		spreadArray("")
		// A missing or implicit table gets a [header] of its own.
		kv := newKeyValue(path[len(path)-1:], node, nl)
		return d.mutate(func() { appendTable(d.cst, path[:len(path)-1], kv, nl) })
	}
}

// newline returns the line ending of the document's first line.
func (d *Document) newline() string {
	text := d.String()
	if i := strings.Index(text, "\n"); i > 0 && text[i-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}

// mutate applies change to the CST and re-evaluates it. If the result is
// invalid, the document is restored.
func (d *Document) mutate(change func()) error {
	before := d.String()
	change()
	root, err := eval.Evaluate(d.cst)
	if err != nil {
		cst, parseErr := parser.Parse(before)
		if parseErr != nil {
			panic(fmt.Sprintf("tomledit: cannot restore document: %v", parseErr))
		}
		d.cst = cst
		return fmt.Errorf("evaluate: %w", err)
	}
	d.root = root
	return nil
}

// get returns the value at keys if it has type T.
//
//nolint:ireturn // T is a concrete value type chosen by the typed getters.
func get[T any](d *Document, keys []string) (T, error) {
	var zero T
	v, ok := d.Get(keys...)
	if !ok {
		return zero, fmt.Errorf("%s: %w", dotted(keys), ErrNotFound)
	}
	typed, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("%s: %w: got %T, want %T", dotted(keys), ErrType, v, zero)
	}
	return typed, nil
}

// dotted renders a key path for error messages.
func dotted(path []string) string { return strings.Join(path, ".") }

// appendTable adds a [path] section holding kv at the end of doc, separated
// from the text before it by a blank line.
func appendTable(doc *ast.Document, path []string, kv *ast.KeyValue, nl string) {
	header := &ast.TableHeader{Key: newKey(path), Trailing: nl}
	if text := doc.String(); text != "" {
		header.Leading = doc.Trailing
		if !strings.HasSuffix(text, "\n") {
			text += nl
			header.Leading += nl
		}
		if !strings.HasSuffix(text, nl+nl) {
			header.Leading += nl
		}
	}
	doc.Trailing = ""
	doc.Tables = append(doc.Tables, &ast.Table{Header: header, Body: []*ast.KeyValue{kv}})
}

// newKey returns the key for path, quoting names that cannot be bare.
func newKey(path []string) *ast.Key {
	key := &ast.Key{}
	for _, name := range path {
		key.Parts = append(key.Parts, format.Key(name))
	}
	return key
}

// newKeyValue returns "k1.k2 = value" followed by newline.
func newKeyValue(path []string, value ast.Value, newline string) *ast.KeyValue {
	key := newKey(path)
	key.Parts[len(key.Parts)-1].Suffix = " "
	ast.ValueDecor(value).Prefix = " "
	return &ast.KeyValue{Key: key, Value: value, Trailing: newline}
}

// sectionIndent returns the indentation of the last key/value of section,
// which a key/value appended to it takes over.
func sectionIndent(section *ast.Table) string {
	if n := len(section.Body); n > 0 {
		return lastLine(section.Body[n-1].Leading)
	}
	return ""
}

// appendKeyValue adds kv to the end of section with the indentation of the
// section's last key/value.
func appendKeyValue(section *ast.Table, kv *ast.KeyValue) {
	if n := len(section.Body); n > 0 {
		last := section.Body[n-1]
		kv.Leading = lastLine(last.Leading)
		last.Trailing = endLine(last.Trailing, kv.Trailing)
	} else if section.Header != nil {
		section.Header.Trailing = endLine(section.Header.Trailing, kv.Trailing)
	}
	section.Body = append(section.Body, kv)
}

// appendEntry adds kv to the end of an inline table, spaced like the entry
// before it, and moves the trivia before "}" so the commas stay balanced.
func appendEntry(table *ast.InlineTable, kv *ast.KeyValue) {
	entry := &ast.InlineEntry{KeyValue: kv}
	n := len(table.Entries)
	if n == 0 {
		kv.Leading, table.Trailing = " ", " "
		table.Entries = []*ast.InlineEntry{entry}
		return
	}
	last := table.Entries[n-1]
	kv.Leading = lastLine(last.KeyValue.Leading)
	if kv.Leading == "" {
		kv.Leading = " "
	}
	if table.TrailingComma {
		entry.AfterComma = lineEnding(last.AfterComma)
	} else {
		addComma(last, kv)
	}
	table.Entries = append(table.Entries, entry)
}

// addComma gives the last entry of an inline table a comma before kv is
// appended after it: the rest of its line goes after the comma, the
// indentation of "}" after kv.
func addComma(last *ast.InlineEntry, kv *ast.KeyValue) {
	moved := last.KeyValue.Trailing
	last.KeyValue.Trailing = ""
	i := strings.Index(moved, "\n")
	if i < 0 {
		kv.Trailing = moved
		return
	}
	last.AfterComma = moved[:i+1]
	kv.Trailing = "\n" + moved[i+1:]
	if strings.HasSuffix(last.AfterComma, "\r\n") {
		kv.Trailing = "\r" + kv.Trailing
	}
}

// lastLine returns the text after the last newline in s.
func lastLine(s string) string { return s[strings.LastIndex(s, "\n")+1:] }

// lineEnding returns the newline that ends s, or "" if s does not end one.
func lineEnding(s string) string {
	switch {
	case strings.HasSuffix(s, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(s, "\n"):
		return "\n"
	}
	return ""
}

// endLine returns trailing trivia that ends with a newline, which it lacks
// only at the end of the document.
func endLine(trailing, newline string) string {
	if strings.HasSuffix(trailing, "\n") {
		return trailing
	}
	return trailing + newline
}

// deletePath removes from doc every table section whose header lies at or
// below path, and every key/value that defines path or a key below it.
func deletePath(doc *ast.Document, path []string) {
	doc.Root.Body = deleteKeyValues(doc.Root.Body, nil, path)
	tables := doc.Tables[:0]
	for _, section := range doc.Tables {
		header := keyNames(section.Header.Key)
		if hasPrefix(header, path) {
			continue
		}
		section.Body = deleteKeyValues(section.Body, header, path)
		tables = append(tables, section)
	}
	doc.Tables = tables
}

// deleteKeyValues removes the key/values of a section at base that define
// path or a key below it.
func deleteKeyValues(body []*ast.KeyValue, base, path []string) []*ast.KeyValue {
	kept := body[:0]
	for _, kv := range body {
		if deleteKeyValue(kv, base, path) {
			continue
		}
		kept = append(kept, kv)
	}
	return kept
}

// deleteKeyValue reports whether kv, found in a table at base, defines path
// or a key below it. If path lies inside kv's inline table, it deletes the
// matching entries from that table instead.
func deleteKeyValue(kv *ast.KeyValue, base, path []string) bool {
	key := append(slices.Clip(base), keyNames(kv.Key)...)
	if hasPrefix(key, path) {
		return true
	}
	if table, ok := kv.Value.(*ast.InlineTable); ok && hasPrefix(path, key) {
		for i, entry := range slices.Backward(table.Entries) {
			if deleteKeyValue(entry.KeyValue, key, path) {
				deleteEntry(table, i)
			}
		}
	}
	return false
}

// deleteEntry removes entry i of an inline table and moves the trivia
// around it so the commas and the spacing inside the braces stay balanced.
func deleteEntry(table *ast.InlineTable, i int) {
	removed := table.Entries[i].KeyValue
	table.Entries = slices.Delete(table.Entries, i, i+1)
	switch {
	case len(table.Entries) == 0:
		table.TrailingComma, table.Trailing = false, ""
	case i == 0:
		// The new first entry takes the removed entry's place after "{",
		// keeping comment lines of its own.
		next := table.Entries[0].KeyValue
		if strings.Contains(next.Leading, "\n") {
			next.Leading = strings.TrimSuffix(removed.Leading, lastLine(removed.Leading)) + next.Leading
		} else {
			next.Leading = removed.Leading
		}
	case i == len(table.Entries) && !table.TrailingComma:
		// The new last entry loses its comma; the rest of its line and the
		// indentation of "}" become the trivia before "}".
		last := table.Entries[i-1]
		if last.AfterComma != "" {
			last.KeyValue.Trailing += last.AfterComma + lastLine(removed.Trailing)
		} else {
			last.KeyValue.Trailing += removed.Trailing
		}
		last.AfterComma = ""
	}
}

// keyNames returns the decoded parts of a key that the evaluator accepted.
func keyNames(key *ast.Key) []string {
	n, err := eval.KeyNames(key)
	if err != nil {
		panic(fmt.Sprintf("tomledit: invalid key in evaluated document: %v", err))
	}
	return n
}

// hasPrefix reports whether the key path s starts with prefix.
func hasPrefix(s, prefix []string) bool {
	return len(s) >= len(prefix) && slices.Equal(s[:len(prefix)], prefix)
}
