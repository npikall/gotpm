package eval

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/token"
)

const (
	quoteLen   = 1 // " or '
	mlQuoteLen = 3 // """ or '''

	escapeChar = 0x1b // \e

	hexByteLen   = 2 // \xHH
	shortUnicode = 4 // \uHHHH
	longUnicode  = 8 // \UHHHHHHHH
)

// hexEscapeLen is the number of hex digits after \x, \u and \U.
var hexEscapeLen = map[byte]int{'x': hexByteLen, 'u': shortUnicode, 'U': longUnicode}

// decodeString returns the value of a string token. The lexer has already
// checked the syntax of escapes; only code point ranges are checked here.
func decodeString(typ token.TokenType, raw string, pos ast.Pos) (string, error) {
	switch typ {
	case token.LITERAL_STRING:
		return raw[quoteLen : len(raw)-quoteLen], nil
	case token.ML_LITERAL_STRING:
		return trimFirstNewline(raw[mlQuoteLen : len(raw)-mlQuoteLen]), nil
	case token.ML_BASIC_STRING:
		return unescape(trimFirstNewline(raw[mlQuoteLen:len(raw)-mlQuoteLen]), pos)
	default: // token.BASIC_STRING
		return unescape(raw[quoteLen:len(raw)-quoteLen], pos)
	}
}

// trimFirstNewline drops a newline directly after the opening delimiter of
// a multi-line string.
func trimFirstNewline(s string) string {
	if rest, ok := strings.CutPrefix(s, "\n"); ok {
		return rest
	}
	if rest, ok := strings.CutPrefix(s, "\r\n"); ok {
		return rest
	}
	return s
}

var simpleEscapes = map[byte]byte{
	'b': '\b', 't': '\t', 'n': '\n', 'f': '\f', 'r': '\r', 'e': escapeChar, '"': '"', '\\': '\\',
}

func unescape(s string, pos ast.Pos) (string, error) {
	if !strings.Contains(s, `\`) {
		return s, nil
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			sb.WriteByte(s[i])
			continue
		}
		i++
		if c, ok := simpleEscapes[s[i]]; ok {
			sb.WriteByte(c)
			continue
		}
		hexLen := hexEscapeLen[s[i]]
		if hexLen == 0 {
			// Line-ending backslash: skip whitespace and newlines up to the
			// next non-whitespace character.
			rest := strings.TrimLeft(s[i:], " \t\r\n")
			i = len(s) - len(rest) - 1
			continue
		}
		code, _ := strconv.ParseInt(s[i+1:i+1+hexLen], 16, 32)
		if code > utf8.MaxRune || !utf8.ValidRune(rune(code)) {
			return "", errorAt(pos, "invalid unicode code point %X in escape", code)
		}
		sb.WriteRune(rune(code))
		i += hexLen
	}
	return sb.String(), nil
}
