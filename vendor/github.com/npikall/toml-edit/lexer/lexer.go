package lexer

import (
	"regexp"
	"unicode/utf8"

	"github.com/npikall/toml-edit/token"
)

// context is an open bracket the lexer is inside of. It decides whether a
// comma or newline switches between key and value position.
type context int

const (
	ctxNone context = iota // top level
	ctxArray
	ctxInlineTable
	ctxTableHeader
	ctxArrayTableHeader
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // line of current char
	col          int  // column of current char

	expectValue bool      // the next token is a value, not a key
	stack       []context // open brackets, innermost last
	strict10    bool      // reject syntax added in TOML 1.1
}

// Option configures a Lexer.
type Option func(*Lexer)

// Strict10 makes the lexer reject syntax that TOML 1.1 added to 1.0: the
// escapes \xHH and \e, and times without seconds. Such tokens are ILLEGAL.
func Strict10() Option {
	return func(l *Lexer) { l.strict10 = true }
}

// New returns a Lexer for TOML 1.1 input, or 1.0 with Strict10.
func New(input string, opts ...Option) *Lexer {
	l := &Lexer{input: input, line: 1}
	for _, opt := range opts {
		opt(l)
	}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	start, line, col := l.position, l.line, l.col

	var tokType token.TokenType
	switch {
	case l.atEOF():
		return token.Token{Type: token.EOF, Line: line, Col: col}
	case isWhitespace(l.ch):
		l.readWhile(isWhitespace)
		tokType = token.WHITESPACE
	case l.ch == '\n', l.ch == '\r' && l.peekChar() == '\n':
		if l.ch == '\r' {
			l.readChar()
		}
		l.readChar()
		l.endLine()
		tokType = token.NEWLINE
	case l.ch == '#':
		tokType = l.readComment()
	case l.ch == '=':
		l.readChar()
		l.expectValue = true
		tokType = token.EQUALS
	case l.expectValue && isScalarChar(l.ch):
		l.readScalar()
		l.expectValue = false
		tokType = l.valueType(l.input[start:l.position])
	case l.ch == '.':
		l.readChar()
		tokType = token.DOT
	case l.ch == ',':
		l.readChar()
		l.expectValue = l.top() == ctxArray
		tokType = token.COMMA
	case l.ch == '[':
		tokType = l.readOpenBracket()
	case l.ch == ']':
		tokType = l.readCloseBracket()
	case l.ch == '{':
		l.readChar()
		if l.expectValue {
			l.push(ctxInlineTable)
			l.expectValue = false
		}
		tokType = token.LBRACE
	case l.ch == '}':
		l.readChar()
		if l.top() == ctxInlineTable {
			l.pop()
		}
		tokType = token.RBRACE
	case l.ch == '"' || l.ch == '\'':
		tokType = l.readString()
		l.expectValue = false
	case IsBareKeyChar(l.ch):
		l.readWhile(IsBareKeyChar)
		tokType = token.BARE_KEY
	default:
		l.readChar()
		tokType = token.ILLEGAL
	}
	return token.Token{Type: tokType, Literal: l.input[start:l.position], Line: line, Col: col}
}

// readComment reads up to the end of the line. A comment containing control
// characters or invalid UTF-8 is ILLEGAL as a whole.
func (l *Lexer) readComment() token.TokenType {
	start := l.position
	l.readWhile(func(ch byte) bool { return ch != '\n' && ch != '\r' })
	comment := l.input[start:l.position]
	for i := range len(comment) {
		if !isStringChar(comment[i]) {
			return token.ILLEGAL
		}
	}
	if !utf8.ValidString(comment) {
		return token.ILLEGAL
	}
	return token.COMMENT
}

// readOpenBracket distinguishes an array ("a = [") from a table header
// ("[table]" or "[[array-of-tables]]"), which only exists at the top level.
func (l *Lexer) readOpenBracket() token.TokenType {
	switch {
	case l.expectValue:
		l.readChar()
		l.push(ctxArray)
		return token.LBRACKET
	case len(l.stack) > 0:
		l.readChar()
		return token.LBRACKET
	case l.peekChar() == '[':
		l.readChar()
		l.readChar()
		l.push(ctxArrayTableHeader)
		return token.DOUBLE_LBRACKET
	default:
		l.readChar()
		l.push(ctxTableHeader)
		return token.LBRACKET
	}
}

func (l *Lexer) readCloseBracket() token.TokenType {
	switch l.top() {
	case ctxArrayTableHeader:
		if l.peekChar() == ']' {
			l.readChar()
			l.readChar()
			l.pop()
			return token.DOUBLE_RBRACKET
		}
	case ctxArray:
		l.pop()
		l.expectValue = false
	case ctxTableHeader:
		l.pop()
	case ctxNone, ctxInlineTable:
	}
	l.readChar()
	return token.RBRACKET
}

// endLine updates the context after a newline. Newlines end a top-level
// expression, and an unterminated table header, but not an array or inline
// table.
func (l *Lexer) endLine() {
	if top := l.top(); top == ctxTableHeader || top == ctxArrayTableHeader {
		l.pop()
	}
	if len(l.stack) == 0 {
		l.expectValue = false
	}
}

func (l *Lexer) push(c context) {
	l.stack = append(l.stack, c)
}

func (l *Lexer) pop() {
	l.stack = l.stack[:len(l.stack)-1]
}

// top returns the innermost open context.
func (l *Lexer) top() context {
	if len(l.stack) == 0 {
		return ctxNone
	}
	return l.stack[len(l.stack)-1]
}

// Read a char from the input and advance the position
func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

// readScalar reads a bare value. A date followed by a space and a digit is
// a date-time with a space delimiter ("1979-05-27 07:32:00"), so the space
// and time are included.
func (l *Lexer) readScalar() {
	start := l.position
	l.readWhile(isScalarChar)
	if l.ch == ' ' && isDigit(l.peekChar()) && reLocalDate.MatchString(l.input[start:l.position]) {
		l.readChar()
		l.readWhile(isScalarChar)
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) atEOF() bool {
	return l.position >= len(l.input)
}

func (l *Lexer) readWhile(pred func(byte) bool) {
	for !l.atEOF() && pred(l.ch) {
		l.readChar()
	}
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// IsBareKeyChar reports whether ch may appear in a bare key.
func IsBareKeyChar(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || '0' <= ch && ch <= '9' || ch == '_' || ch == '-'
}

// isScalarChar reports whether ch can be part of a bare value such as a
// number, boolean or date.
func isScalarChar(ch byte) bool {
	return IsBareKeyChar(ch) || ch == '+' || ch == '.' || ch == ':'
}

// Patterns follow the ABNF in the TOML v1.1.0 spec. Range checks (month 13,
// hour 25, integer overflow) are semantic and happen in the evaluator.
const (
	decInt      = `[+-]?(?:0|[1-9](?:_?[0-9])*)`
	zeroPrefInt = `[0-9](?:_?[0-9])*`
	fullDate    = `[0-9]{4}-[0-9]{2}-[0-9]{2}`
	partialTime = `[0-9]{2}:[0-9]{2}(?::[0-9]{2}(?:\.[0-9]+)?)?`
	timeOffset  = `(?:[Zz]|[+-][0-9]{2}:[0-9]{2})`
)

var (
	reInteger = regexp.MustCompile(`^(?:` + decInt +
		`|0x[0-9A-Fa-f](?:_?[0-9A-Fa-f])*` +
		`|0o[0-7](?:_?[0-7])*` +
		`|0b[01](?:_?[01])*)$`)
	reFloat = regexp.MustCompile(`^(?:` + decInt + `(?:\.` + zeroPrefInt + `)?[eE][+-]?` + zeroPrefInt +
		`|` + decInt + `\.` + zeroPrefInt +
		`|[+-]?(?:inf|nan))$`)
	reLocalDate      = regexp.MustCompile(`^` + fullDate + `$`)
	reLocalTime      = regexp.MustCompile(`^` + partialTime + `$`)
	reLocalDateTime  = regexp.MustCompile(`^` + fullDate + `[Tt ]` + partialTime + `$`)
	reOffsetDateTime = regexp.MustCompile(`^` + fullDate + `[Tt ]` + partialTime + timeOffset + `$`)
	// reSeconds matches the time part of a time or date-time if it has
	// seconds. An offset ("+07:00") never matches, as it follows no colon.
	reSeconds = regexp.MustCompile(`[0-9]{2}:[0-9]{2}:[0-9]{2}`)
)

// valueType returns the token type of a bare value, which is ILLEGAL if it
// is no valid value.
func (l *Lexer) valueType(lit string) token.TokenType {
	typ := classifyScalar(lit)
	switch typ {
	case token.LOCAL_TIME, token.LOCAL_DATETIME, token.OFFSET_DATETIME:
		if l.strict10 && !reSeconds.MatchString(lit) {
			return token.ILLEGAL
		}
	}
	return typ
}

func classifyScalar(lit string) token.TokenType {
	switch {
	case lit == "true" || lit == "false":
		return token.BOOL
	case reInteger.MatchString(lit):
		return token.INTEGER
	case reFloat.MatchString(lit):
		return token.FLOAT
	case reLocalDate.MatchString(lit):
		return token.LOCAL_DATE
	case reLocalTime.MatchString(lit):
		return token.LOCAL_TIME
	case reLocalDateTime.MatchString(lit):
		return token.LOCAL_DATETIME
	case reOffsetDateTime.MatchString(lit):
		return token.OFFSET_DATETIME
	default:
		return token.ILLEGAL
	}
}
