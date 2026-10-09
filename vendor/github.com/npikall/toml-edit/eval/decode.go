package eval

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/token"
)

func decodeValue(v ast.Value) (any, error) {
	switch v := v.(type) {
	case *ast.Scalar:
		return decodeScalar(v)
	case *ast.Array:
		out := make([]any, len(v.Items))
		for i, item := range v.Items {
			var err error
			if out[i], err = decodeValue(item.Value); err != nil {
				return nil, err
			}
		}
		return out, nil
	case *ast.InlineTable:
		t := newTable(kindInline)
		t.inline = v
		for _, entry := range v.Entries {
			if err := insertKeyValue(t, entry.KeyValue); err != nil {
				return nil, err
			}
		}
		return t, nil
	default:
		panic(fmt.Sprintf("eval: unexpected value node %T", v))
	}
}

func decodeScalar(s *ast.Scalar) (any, error) {
	switch s.Type {
	case token.BOOL:
		return s.Raw == "true", nil
	case token.BASIC_STRING, token.LITERAL_STRING, token.ML_BASIC_STRING, token.ML_LITERAL_STRING:
		return decodeString(s.Type, s.Raw, s.Pos)
	case token.INTEGER:
		return decodeInteger(s)
	case token.FLOAT:
		return decodeFloat(s)
	default: // token.OFFSET_DATETIME, LOCAL_DATETIME, LOCAL_DATE or LOCAL_TIME
		return decodeDateTime(s)
	}
}

// decodeInteger uses base 0, which accepts the 0x, 0o and 0b prefixes and
// underscores. The lexer already rejected the spellings Go allows but TOML
// does not (leading zeros, "0x_1", signed prefixes).
func decodeInteger(s *ast.Scalar) (int64, error) {
	n, err := strconv.ParseInt(s.Raw, 0, 64)
	if err != nil {
		return 0, errorAt(s.Pos, "integer out of range")
	}
	return n, nil
}

// decodeFloat relies on strconv accepting TOML's spellings of inf; nan may
// carry a sign in TOML but not in strconv.
func decodeFloat(s *ast.Scalar) (float64, error) {
	if strings.TrimLeft(s.Raw, "+-") == "nan" {
		return math.NaN(), nil
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s.Raw, "_", ""), 64)
	if err != nil {
		return 0, errorAt(s.Pos, "float out of range")
	}
	return f, nil
}
