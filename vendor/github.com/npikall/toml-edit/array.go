package tomledit

import (
	"strings"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/format"
)

// arrayLayout is the line structure of a multi-line array: the rest of the
// line after "[", the indentation of each element, the trivia before "]" and
// the comments of each element, keyed by its value.
type arrayLayout struct {
	open    string
	indent  string
	newline string
	closing string
	notes   map[string][]elementNote
}

// elementNote is an element of an old array with the comment lines above it
// and the rest of its line after the comma.
type elementNote struct {
	value    ast.Value
	comments string
	after    string
}

// newLayout is the layout of a new multi-line array whose key is indented by
// indent: elements two spaces deeper, "]" in the key's column.
func newLayout(indent, newline string) arrayLayout {
	return arrayLayout{open: newline, indent: indent + "  ", newline: newline, closing: indent}
}

// multiline reports whether the array is written across several lines.
func multiline(arr *ast.Array) bool {
	var sb strings.Builder
	sb.WriteString(arr.Trailing)
	for _, item := range arr.Items {
		d := ast.ValueDecor(item.Value)
		sb.WriteString(d.Prefix + d.Suffix + item.AfterComma)
	}
	return strings.Contains(sb.String(), "\n")
}

// layoutOf reads the layout of a multi-line array whose elements decode to
// values. Elements sharing a line are spread one per line; newline is the
// document's line ending, used where the array has none of its own.
func layoutOf(arr *ast.Array, values []any, newline string) arrayLayout {
	layout := arrayLayout{open: newline, indent: "  ", newline: newline, notes: map[string][]elementNote{}}
	if len(arr.Items) == 0 {
		open, closing, _ := strings.Cut(arr.Trailing, "\n")
		layout.open, layout.closing = open+"\n", closing
		layout.newline = lineEnding(layout.open)
		return layout
	}
	first := ast.ValueDecor(arr.Items[0].Value).Prefix
	if i := strings.Index(first, "\n"); i >= 0 {
		layout.open = first[:i+1]
		layout.newline = lineEnding(layout.open)
	}
	if i := firstOnOwnLine(arr); i >= 0 {
		layout.indent = lastLine(ast.ValueDecor(arr.Items[i].Value).Prefix)
	}
	if arr.TrailingComma {
		layout.closing = arr.Trailing
	}
	last := len(arr.Items) - 1
	for i, item := range arr.Items {
		d := ast.ValueDecor(item.Value)
		prefix := d.Prefix
		if i == 0 {
			prefix = strings.TrimPrefix(prefix, layout.open)
		}
		note := elementNote{
			value:    item.Value,
			comments: strings.TrimSuffix(prefix, lastLine(prefix)),
			after:    item.AfterComma,
		}
		// A comment between the value and its comma moves after the comma.
		if j := strings.Index(d.Suffix, "\n"); j >= 0 {
			note.after = d.Suffix[:j+1]
			if i == last && !arr.TrailingComma {
				layout.closing = d.Suffix[j+1:]
			}
		}
		if i < len(values) {
			if key, ok := canonical(values[i]); ok {
				layout.notes[key] = append(layout.notes[key], note)
			}
		}
	}
	return layout
}

// firstOnOwnLine returns the index of the first element of arr that starts a
// line, or -1 if every element shares the line of "[".
func firstOnOwnLine(arr *ast.Array) int {
	for i, item := range arr.Items {
		if strings.Contains(ast.ValueDecor(item.Value).Prefix, "\n") {
			return i
		}
		if i > 0 {
			prev := arr.Items[i-1]
			if strings.Contains(ast.ValueDecor(prev.Value).Suffix+prev.AfterComma, "\n") {
				return i
			}
		}
	}
	return -1
}

// canonical returns the text a new value v would be written as.
func canonical(v any) (string, bool) {
	node, err := format.Value(v)
	if err != nil {
		return "", false
	}
	return ast.Raw(node), true
}

// shallowCopy returns a copy of v whose decor can change without touching v,
// which stays in the old tree in case the edit is rolled back.
//
//nolint:ireturn // ast.Value is a closed sum of Scalar, Array and InlineTable.
func shallowCopy(v ast.Value) ast.Value {
	switch v := v.(type) {
	case *ast.Scalar:
		c := *v
		return &c
	case *ast.Array:
		c := *v
		return &c
	case *ast.InlineTable:
		c := *v
		return &c
	}
	return v
}

// apply writes arr one element per line in layout, each followed by a comma.
// An element equal to one of the old array keeps that element's spelling and
// comments. It consumes the comments, so a layout is applied only once.
func (layout *arrayLayout) apply(arr *ast.Array) {
	if len(arr.Items) == 0 {
		return
	}
	for i, item := range arr.Items {
		comments, after := "", layout.newline
		key := ast.Raw(item.Value)
		if notes := layout.notes[key]; len(notes) > 0 {
			note := notes[0]
			layout.notes[key] = notes[1:]
			item.Value, comments = shallowCopy(note.value), note.comments
			if strings.HasSuffix(note.after, "\n") {
				after = note.after
			}
		}
		d := ast.ValueDecor(item.Value)
		d.Prefix, d.Suffix = comments+layout.indent, ""
		if i == 0 {
			d.Prefix = layout.open + d.Prefix
		}
		item.AfterComma = after
	}
	arr.TrailingComma = true
	arr.Trailing = layout.closing
}
