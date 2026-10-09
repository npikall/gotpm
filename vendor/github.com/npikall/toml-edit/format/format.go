// Package format turns Go values into CST nodes with canonical TOML text. It
// is used for new and changed nodes only; parsed nodes keep their source
// text.
package format

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/lexer"
	"github.com/npikall/toml-edit/token"
)

// ErrUnsupported is returned for Go values that have no TOML representation.
var ErrUnsupported = errors.New("unsupported value")

// Value returns the CST node for v. Supported are strings, booleans, all
// integer and float types, time.Time, eval.LocalDate, eval.LocalTime,
// eval.LocalDateTime, slices and arrays of those (as arrays), and maps with
// string keys (as inline tables, keys sorted).
//
//nolint:ireturn // ast.Value is a closed sum of Scalar, Array and InlineTable.
func Value(v any) (ast.Value, error) {
	switch v := v.(type) {
	case string:
		if !utf8.ValidString(v) {
			return nil, fmt.Errorf("%w: string %q is not valid UTF-8", ErrUnsupported, v)
		}
		return &ast.Scalar{Type: token.BASIC_STRING, Raw: quote(v)}, nil
	case bool:
		return &ast.Scalar{Type: token.BOOL, Raw: strconv.FormatBool(v)}, nil
	case time.Time:
		return offsetDateTime(v)
	case eval.LocalDate:
		return &ast.Scalar{Type: token.LOCAL_DATE, Raw: v.String()}, nil
	case eval.LocalTime:
		return &ast.Scalar{Type: token.LOCAL_TIME, Raw: v.String()}, nil
	case eval.LocalDateTime:
		return &ast.Scalar{Type: token.LOCAL_DATETIME, Raw: v.String()}, nil
	}
	rv := reflect.ValueOf(v)
	switch {
	case rv.CanInt():
		return &ast.Scalar{Type: token.INTEGER, Raw: strconv.FormatInt(rv.Int(), 10)}, nil
	case rv.CanUint():
		if rv.Uint() > math.MaxInt64 {
			return nil, fmt.Errorf("%w: integer %d overflows int64", ErrUnsupported, rv.Uint())
		}
		return &ast.Scalar{Type: token.INTEGER, Raw: strconv.FormatUint(rv.Uint(), 10)}, nil
	case rv.CanFloat():
		return &ast.Scalar{Type: token.FLOAT, Raw: formatFloat(rv.Float(), rv.Type().Bits())}, nil
	case rv.Kind() == reflect.Slice, rv.Kind() == reflect.Array:
		return array(rv)
	case rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String:
		return inlineTable(rv)
	}
	return nil, fmt.Errorf("%w: %T", ErrUnsupported, v)
}

// Key returns a key part for name: bare if possible, else a basic string.
func Key(name string) *ast.KeyPart {
	if isBare(name) {
		return &ast.KeyPart{Type: token.BARE_KEY, Raw: name}
	}
	return &ast.KeyPart{Type: token.BASIC_STRING, Raw: quote(name)}
}

// isBare reports whether name can be written as a bare key.
func isBare(name string) bool {
	for i := range len(name) {
		if !lexer.IsBareKeyChar(name[i]) {
			return false
		}
	}
	return name != ""
}

const (
	maxYear       = 9999
	secondsPerMin = 60
)

// offsetDateTime formats t as RFC 3339, which allows neither years beyond
// four digits nor offsets with seconds.
func offsetDateTime(t time.Time) (*ast.Scalar, error) {
	if _, offset := t.Zone(); t.Year() < 0 || t.Year() > maxYear || offset%secondsPerMin != 0 {
		return nil, fmt.Errorf("%w: time %s has no RFC 3339 form", ErrUnsupported, t)
	}
	return &ast.Scalar{Type: token.OFFSET_DATETIME, Raw: t.Format(time.RFC3339Nano)}, nil
}

// array formats a slice as "[a, b, c]".
func array(rv reflect.Value) (*ast.Array, error) {
	arr := &ast.Array{}
	for i := range rv.Len() {
		value, err := Value(rv.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		if i > 0 {
			ast.ValueDecor(value).Prefix = " "
		}
		arr.Items = append(arr.Items, &ast.ArrayItem{Value: value})
	}
	return arr, nil
}

// inlineTable formats a map as "{ k1 = v1, k2 = v2 }" with sorted keys.
func inlineTable(rv reflect.Value) (*ast.InlineTable, error) {
	table := &ast.InlineTable{}
	keys := rv.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
	for _, k := range keys {
		value, err := Value(rv.MapIndex(k).Interface())
		if err != nil {
			return nil, err
		}
		part := Key(k.String())
		part.Prefix, part.Suffix = " ", " "
		ast.ValueDecor(value).Prefix = " "
		table.Entries = append(table.Entries, &ast.InlineEntry{
			KeyValue: &ast.KeyValue{Key: &ast.Key{Parts: []*ast.KeyPart{part}}, Value: value},
		})
	}
	if len(table.Entries) > 0 {
		table.Trailing = " "
	}
	return table, nil
}

// formatFloat returns the shortest text that parses back to f, always with
// a fraction or exponent so it does not read as an integer.
func formatFloat(f float64, bitSize int) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	s := strconv.FormatFloat(f, 'g', -1, bitSize)
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

const delChar = 0x7f

var escapes = map[rune]string{
	'\b': `\b`, '\t': `\t`, '\n': `\n`, '\f': `\f`, '\r': `\r`, '"': `\"`, '\\': `\\`,
}

// quote returns s as a basic string, escaping quotes, backslashes and
// control characters. It uses no escapes added in TOML 1.1 (\e, \xHH), so
// the result is valid TOML 1.0 too.
func quote(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch esc, ok := escapes[r]; {
		case ok:
			sb.WriteString(esc)
		case r < ' ' || r == delChar: // control characters
			fmt.Fprintf(&sb, `\u%04X`, r)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}
