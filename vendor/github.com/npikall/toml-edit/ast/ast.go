// Package ast defines the concrete syntax tree (CST) of a TOML document.
//
// Every node keeps its raw source text and the trivia (whitespace, comments
// and newlines) around it, so String reproduces the input byte for byte.
//
// Trivia rules: comments and blank lines before a node, including the
// indentation of its line, are the node's Leading trivia. Whitespace and a
// comment after a node on the same line, including the newline, are its
// Trailing trivia. Trivia at the end of the document that belongs to no node
// is the Document's Trailing trivia.
package ast

import (
	"strings"

	"github.com/npikall/toml-edit/token"
)

// Document is a whole TOML file. Root holds the key/values before the first
// table header; Tables holds every [table] and [[array-of-tables]] section in
// source order.
type Document struct {
	Root     *Table
	Tables   []*Table
	Trailing string
}

func (d *Document) String() string {
	var sb strings.Builder
	d.Root.write(&sb)
	for _, t := range d.Tables {
		t.write(&sb)
	}
	sb.WriteString(d.Trailing)
	return sb.String()
}

// Table is a table section: a header followed by its key/values. The root
// table has no header.
type Table struct {
	Header *TableHeader // nil for the root table
	Body   []*KeyValue
}

func (t *Table) write(sb *strings.Builder) {
	if t.Header != nil {
		t.Header.write(sb)
	}
	for _, kv := range t.Body {
		kv.write(sb)
	}
}

// TableHeader is "[key]" or, if ArrayOfTables is set, "[[key]]". Whitespace
// inside the brackets is part of the key's decor.
type TableHeader struct {
	Leading       string
	ArrayOfTables bool
	Key           *Key
	Trailing      string
}

func (h *TableHeader) write(sb *strings.Builder) {
	open, closing := token.LBRACKET, token.RBRACKET
	if h.ArrayOfTables {
		open, closing = token.DOUBLE_LBRACKET, token.DOUBLE_RBRACKET
	}
	sb.WriteString(h.Leading)
	sb.WriteString(open)
	h.Key.write(sb)
	sb.WriteString(closing)
	sb.WriteString(h.Trailing)
}

// KeyValue is "key = value". Whitespace before "=" is the suffix of the
// key's last part, whitespace after it is the value's prefix. Inside an
// inline table, Trailing is the trivia between the value and the next "," or
// "}".
type KeyValue struct {
	Leading  string
	Key      *Key
	Value    Value
	Trailing string
}

func (kv *KeyValue) write(sb *strings.Builder) {
	sb.WriteString(kv.Leading)
	kv.Key.write(sb)
	sb.WriteString(token.EQUALS)
	writeValue(sb, kv.Value)
	sb.WriteString(kv.Trailing)
}

// Key is a simple or dotted key. The dots between parts are implicit.
type Key struct {
	Parts []*KeyPart
}

func (k *Key) write(sb *strings.Builder) {
	for i, p := range k.Parts {
		if i > 0 {
			sb.WriteString(token.DOT)
		}
		sb.WriteString(p.Prefix)
		sb.WriteString(p.Raw)
		sb.WriteString(p.Suffix)
	}
}

// KeyPart is one simple key: a bare key or a quoted (basic or literal) key.
// Raw is the source text, including quotes.
type KeyPart struct {
	Decor
	Type token.TokenType
	Raw  string
	Pos  Pos
}

// Pos is a 1-based source position (Col counts bytes). It is the zero value
// for nodes that were not parsed from source.
type Pos struct {
	Line int
	Col  int
}

// Decor is the whitespace (and, inside arrays and inline tables, comments
// and newlines) directly around a key part or value.
type Decor struct {
	Prefix string
	Suffix string
}

func (d *Decor) decor() *Decor { return d }

// Value is a Scalar, an Array or an InlineTable.
type Value interface {
	decor() *Decor
	writeRaw(sb *strings.Builder)
}

// ValueDecor returns the decor of v so callers can adjust its whitespace.
func ValueDecor(v Value) *Decor { return v.decor() }

// Raw returns the text of v without its decor.
func Raw(v Value) string {
	var sb strings.Builder
	v.writeRaw(&sb)
	return sb.String()
}

func writeValue(sb *strings.Builder, v Value) {
	d := v.decor()
	sb.WriteString(d.Prefix)
	v.writeRaw(sb)
	sb.WriteString(d.Suffix)
}

// Scalar is a string, number, boolean or date/time. Raw is the exact source
// text; decoding happens in the evaluator.
type Scalar struct {
	Decor
	Type token.TokenType
	Raw  string
	Pos  Pos
}

func (s *Scalar) writeRaw(sb *strings.Builder) { sb.WriteString(s.Raw) }

// Array is "[v1, v2]". Each value's decor holds the trivia between it and the
// surrounding brackets or commas. Trailing is the trivia before "]" that
// follows the trailing comma, or the whole inside of an empty array.
type Array struct {
	Decor
	Items         []*ArrayItem
	TrailingComma bool
	Trailing      string
}

// ArrayItem is one array element. AfterComma is the rest of the line after
// the element's comma (whitespace, an optional comment and the newline), so a
// comment after the comma stays with the element it describes. It is empty
// unless the line ends there.
type ArrayItem struct {
	Value      Value
	AfterComma string
}

func (a *Array) writeRaw(sb *strings.Builder) {
	sb.WriteString(token.LBRACKET)
	for i, item := range a.Items {
		writeValue(sb, item.Value)
		if i < len(a.Items)-1 || a.TrailingComma {
			sb.WriteString(token.COMMA)
			sb.WriteString(item.AfterComma)
		}
	}
	sb.WriteString(a.Trailing)
	sb.WriteString(token.RBRACKET)
}

// InlineTable is "{k1 = v1, k2 = v2}". Each entry's Leading and Trailing hold
// the trivia between it and the surrounding braces or commas. Trailing is the
// trivia before "}" that follows the trailing comma, or the whole inside of
// an empty table.
type InlineTable struct {
	Decor
	Entries       []*InlineEntry
	TrailingComma bool
	Trailing      string
}

// InlineEntry is one key/value of an inline table. AfterComma works as in
// ArrayItem.
type InlineEntry struct {
	KeyValue   *KeyValue
	AfterComma string
}

func (t *InlineTable) writeRaw(sb *strings.Builder) {
	sb.WriteString(token.LBRACE)
	for i, entry := range t.Entries {
		entry.KeyValue.write(sb)
		if i < len(t.Entries)-1 || t.TrailingComma {
			sb.WriteString(token.COMMA)
			sb.WriteString(entry.AfterComma)
		}
	}
	sb.WriteString(t.Trailing)
	sb.WriteString(token.RBRACE)
}
