// Package eval turns a CST into semantic values and enforces the TOML rules
// the parser cannot check: duplicate keys, table redefinition, value ranges.
package eval

import (
	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/token"
)

// tableKind records how a table came into existence, which decides how it
// may be extended later.
type tableKind int

const (
	// kindImplicit tables are parents of a [header]; they may still be
	// defined by a header of their own, once.
	kindImplicit tableKind = iota
	// kindExplicit tables were defined by a [header], or are an element of
	// an [[array of tables]].
	kindExplicit
	// kindDotted tables were created by a dotted key; more dotted keys may
	// extend them, headers may not.
	kindDotted
	// kindInline tables are closed once their braces end.
	kindInline
)

// Table is a TOML table with keys in definition order.
type Table struct {
	keys    []string
	entries map[string]any
	kind    tableKind
	aot     map[string]bool          // keys holding an [[array of tables]]
	nodes   map[string]*ast.KeyValue // keys defined by a key/value
	section *ast.Table               // set for the root and [header] tables
	inline  *ast.InlineTable         // set for inline tables
}

func newTable(kind tableKind) *Table {
	return &Table{entries: map[string]any{}, kind: kind, nodes: map[string]*ast.KeyValue{}}
}

// Keys returns the table's keys in the order they were defined.
func (t *Table) Keys() []string { return t.keys }

// Get returns the value of key: a string, int64, float64, bool, time.Time,
// LocalDateTime, LocalDate, LocalTime, []any or *Table.
func (t *Table) Get(key string) (any, bool) {
	v, ok := t.entries[key]
	return v, ok
}

// KeyValue returns the CST node that defines key. It reports false for keys
// that hold a table defined by a header or a dotted key, or an array of
// tables.
func (t *Table) KeyValue(key string) (*ast.KeyValue, bool) {
	kv, ok := t.nodes[key]
	return kv, ok
}

// Section returns the CST section whose body holds t's key/values: the root
// section, or the section of the [header] or [[header]] that defined t. It
// is nil for tables defined implicitly, by dotted keys or inline.
func (t *Table) Section() *ast.Table { return t.section }

// Inline returns the CST node of an inline table. It is nil for other
// tables.
func (t *Table) Inline() *ast.InlineTable { return t.inline }

// Dotted reports whether t was defined by dotted keys.
func (t *Table) Dotted() bool { return t.kind == kindDotted }

func (t *Table) set(key string, v any) {
	if _, ok := t.entries[key]; !ok {
		t.keys = append(t.keys, key)
	}
	t.entries[key] = v
}

// Evaluate builds the table tree of doc and validates it.
func Evaluate(doc *ast.Document) (*Table, error) {
	root := newTable(kindExplicit)
	root.section = doc.Root
	if err := insertBody(root, doc.Root.Body); err != nil {
		return nil, err
	}
	for _, section := range doc.Tables {
		t, err := defineTable(root, section.Header)
		if err != nil {
			return nil, err
		}
		t.section = section
		if err := insertBody(t, section.Body); err != nil {
			return nil, err
		}
	}
	return root, nil
}

// defineTable returns the table that a [header] or [[header]] opens,
// creating implicit parent tables on the way.
func defineTable(root *Table, header *ast.TableHeader) (*Table, error) {
	parts := header.Key.Parts
	t := root
	for _, part := range parts[:len(parts)-1] {
		var err error
		if t, err = descendHeader(t, part); err != nil {
			return nil, err
		}
	}
	last := parts[len(parts)-1]
	name, err := keyName(last)
	if err != nil {
		return nil, err
	}
	existing, exists := t.entries[name]
	if header.ArrayOfTables {
		if exists && !t.aot[name] {
			return nil, errorAt(last.Pos, "cannot define array of tables %q: key already defined", name)
		}
		element := newTable(kindExplicit)
		elements, _ := existing.([]any)
		t.set(name, append(elements, element))
		if t.aot == nil {
			t.aot = map[string]bool{}
		}
		t.aot[name] = true
		return element, nil
	}
	if !exists {
		child := newTable(kindExplicit)
		t.set(name, child)
		return child, nil
	}
	if child, ok := existing.(*Table); ok && child.kind == kindImplicit {
		child.kind = kindExplicit
		return child, nil
	}
	return nil, errorAt(last.Pos, "table %q already defined", name)
}

// descendHeader returns the table named by one non-final part of a header
// key, creating an implicit table if needed. Through an array of tables it
// descends into the last element.
func descendHeader(t *Table, part *ast.KeyPart) (*Table, error) {
	name, err := keyName(part)
	if err != nil {
		return nil, err
	}
	switch v := t.entries[name].(type) {
	case nil:
		child := newTable(kindImplicit)
		t.set(name, child)
		return child, nil
	case *Table:
		if v.kind == kindInline {
			return nil, errorAt(part.Pos, "cannot extend inline table %q", name)
		}
		return v, nil
	case []any:
		if t.aot[name] {
			return v[len(v)-1].(*Table), nil
		}
		return nil, errorAt(part.Pos, "cannot extend static array %q", name)
	default:
		return nil, errorAt(part.Pos, "key %q is already defined as a value", name)
	}
}

func insertBody(t *Table, body []*ast.KeyValue) error {
	for _, kv := range body {
		if err := insertKeyValue(t, kv); err != nil {
			return err
		}
	}
	return nil
}

// insertKeyValue adds kv to t, creating tables for the dotted parts of its
// key.
func insertKeyValue(t *Table, kv *ast.KeyValue) error {
	parts := kv.Key.Parts
	for _, part := range parts[:len(parts)-1] {
		var err error
		if t, err = descendDotted(t, part); err != nil {
			return err
		}
	}
	last := parts[len(parts)-1]
	name, err := keyName(last)
	if err != nil {
		return err
	}
	if _, exists := t.entries[name]; exists {
		return errorAt(last.Pos, "duplicate key %q", name)
	}
	v, err := decodeValue(kv.Value)
	if err != nil {
		return err
	}
	t.set(name, v)
	t.nodes[name] = kv
	return nil
}

// descendDotted returns the table named by one dotted-key part, creating it
// if needed. Tables defined by a header or inline cannot be extended this
// way.
func descendDotted(t *Table, part *ast.KeyPart) (*Table, error) {
	name, err := keyName(part)
	if err != nil {
		return nil, err
	}
	switch v := t.entries[name].(type) {
	case nil:
		child := newTable(kindDotted)
		t.set(name, child)
		return child, nil
	case *Table:
		switch v.kind {
		case kindDotted:
			return v, nil
		case kindImplicit:
			// Not defined yet, so the dotted key defines it.
			v.kind = kindDotted
			return v, nil
		case kindExplicit, kindInline:
		}
		return nil, errorAt(part.Pos, "cannot add keys to table %q with a dotted key", name)
	default:
		return nil, errorAt(part.Pos, "key %q is already defined as a value", name)
	}
}

// KeyNames returns the decoded names of the parts of key.
func KeyNames(key *ast.Key) ([]string, error) {
	names := make([]string, len(key.Parts))
	for i, part := range key.Parts {
		var err error
		if names[i], err = keyName(part); err != nil {
			return nil, err
		}
	}
	return names, nil
}

func keyName(part *ast.KeyPart) (string, error) {
	if part.Type == token.BARE_KEY {
		return part.Raw, nil
	}
	return decodeString(part.Type, part.Raw, part.Pos)
}
