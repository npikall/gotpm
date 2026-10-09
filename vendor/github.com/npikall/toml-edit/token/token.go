package token

type TokenType string

// Token is a single lexeme. Literal is always the exact source text, so
// concatenating the Literals of all tokens reproduces the input.
type Token struct {
	Type    TokenType
	Literal string
	Line    int // 1-based
	Col     int // 1-based, counted in bytes
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Trivia
	WHITESPACE = "WHITESPACE" // spaces and tabs
	NEWLINE    = "NEWLINE"    // \n or \r\n
	COMMENT    = "COMMENT"    // # up to end of line

	// Keys
	BARE_KEY = "BARE_KEY" // A-Za-z0-9_-

	// Strings
	BASIC_STRING      = "BASIC_STRING"      // "..."
	LITERAL_STRING    = "LITERAL_STRING"    // '...'
	ML_BASIC_STRING   = "ML_BASIC_STRING"   // """..."""
	ML_LITERAL_STRING = "ML_LITERAL_STRING" // '''...'''

	// Scalars
	INTEGER         = "INTEGER"
	FLOAT           = "FLOAT"
	BOOL            = "BOOL"
	OFFSET_DATETIME = "OFFSET_DATETIME"
	LOCAL_DATETIME  = "LOCAL_DATETIME"
	LOCAL_DATE      = "LOCAL_DATE"
	LOCAL_TIME      = "LOCAL_TIME"

	// Delimiters
	DOT    = "."
	EQUALS = "="
	COMMA  = ","

	LBRACKET        = "["
	RBRACKET        = "]"
	DOUBLE_LBRACKET = "[["
	DOUBLE_RBRACKET = "]]"
	LBRACE          = "{"
	RBRACE          = "}"
)
