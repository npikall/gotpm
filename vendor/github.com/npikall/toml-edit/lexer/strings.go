package lexer

import (
	"strings"
	"unicode/utf8"

	"github.com/npikall/toml-edit/token"
)

const (
	delimLen       = 3 // """ or '''
	maxQuoteRunLen = 5 // up to two quotes may precede the closing delimiter

	hexEscapeLen       = 2 // \xHH
	shortUnicodeEscLen = 4 // \uHHHH
	longUnicodeEscLen  = 8 // \UHHHHHHHH
)

// readString reads any of the four string kinds starting at the opening
// quote. Invalid strings are still consumed (up to the closing delimiter, or
// up to the end of the line if unterminated) and reported as ILLEGAL.
func (l *Lexer) readString() token.TokenType {
	start := l.position
	quote := l.ch
	basic := quote == '"'
	delim := strings.Repeat(string(quote), delimLen)

	var tokType token.TokenType
	switch {
	case strings.HasPrefix(l.input[l.position:], delim) && basic:
		tokType = l.readMultilineString(quote, true, token.ML_BASIC_STRING)
	case strings.HasPrefix(l.input[l.position:], delim):
		tokType = l.readMultilineString(quote, false, token.ML_LITERAL_STRING)
	case basic:
		tokType = l.readSingleLineString(quote, true, token.BASIC_STRING)
	default:
		tokType = l.readSingleLineString(quote, false, token.LITERAL_STRING)
	}
	if !utf8.ValidString(l.input[start:l.position]) {
		return token.ILLEGAL
	}
	return tokType
}

func (l *Lexer) readSingleLineString(quote byte, escapes bool, valid token.TokenType) token.TokenType {
	l.readChar() // opening quote
	ok := true
	for {
		switch {
		case l.atEOF() || l.ch == '\n' || l.ch == '\r':
			return token.ILLEGAL // unterminated
		case l.ch == quote:
			l.readChar()
			if !ok {
				return token.ILLEGAL
			}
			return valid
		case escapes && l.ch == '\\':
			ok = l.readEscape() && ok
		default:
			ok = isStringChar(l.ch) && ok
			l.readChar()
		}
	}
}

func (l *Lexer) readMultilineString(quote byte, escapes bool, valid token.TokenType) token.TokenType {
	for range delimLen { // opening delimiter
		l.readChar()
	}
	ok := true
	for {
		switch {
		case l.atEOF():
			return token.ILLEGAL // unterminated
		case l.ch == quote:
			// Up to two quotes may directly precede the closing delimiter.
			n := 0
			for l.ch == quote && !l.atEOF() {
				l.readChar()
				n++
			}
			if n >= delimLen {
				if !ok || n > maxQuoteRunLen {
					return token.ILLEGAL
				}
				return valid
			}
		case l.ch == '\n':
			l.readChar()
		case l.ch == '\r':
			ok = l.peekChar() == '\n' && ok
			l.readChar()
		case escapes && l.ch == '\\':
			ok = l.readMultilineEscape() && ok
		default:
			ok = isStringChar(l.ch) && ok
			l.readChar()
		}
	}
}

// readMultilineEscape handles a line-ending backslash, which trims the
// newline and all following whitespace, or falls back to a normal escape.
func (l *Lexer) readMultilineEscape() bool {
	rest := strings.TrimLeft(l.input[l.readPosition:], " \t") // same set as isWhitespace
	if !strings.HasPrefix(rest, "\n") && !strings.HasPrefix(rest, "\r\n") {
		return l.readEscape()
	}
	l.readChar() // backslash
	// A lone CR is left for the caller to reject.
	l.readWhile(func(ch byte) bool {
		return isWhitespace(ch) || ch == '\n' || ch == '\r' && l.peekChar() == '\n'
	})
	return true
}

// readEscape reads a backslash escape sequence and reports whether it is
// valid. Code point range checks for \u and \U happen in the evaluator.
func (l *Lexer) readEscape() bool {
	l.readChar() // backslash
	switch l.ch {
	case '"', '\\', 'b', 'f', 'n', 'r', 't':
		l.readChar()
		return true
	case 'e':
		l.readChar()
		return !l.strict10
	case 'x':
		return l.readHexDigits(hexEscapeLen) && !l.strict10
	case 'u':
		return l.readHexDigits(shortUnicodeEscLen)
	case 'U':
		return l.readHexDigits(longUnicodeEscLen)
	default:
		return false
	}
}

func (l *Lexer) readHexDigits(n int) bool {
	l.readChar() // x, u or U
	for range n {
		if l.atEOF() || !isHexDigit(l.ch) {
			return false
		}
		l.readChar()
	}
	return true
}

// isStringChar reports whether ch may appear unescaped in a string or
// comment: tab, printable ASCII, or any non-ASCII byte (UTF-8 validity is
// checked separately).
func isStringChar(ch byte) bool {
	return ch == '\t' || 0x20 <= ch && ch < 0x7f || ch >= 0x80
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || 'a' <= ch && ch <= 'f' || 'A' <= ch && ch <= 'F'
}
