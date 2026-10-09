// Package parser turns TOML source into a lossless concrete syntax tree.
//
// The parser checks syntax only. Semantic rules such as duplicate keys or
// table redefinition are enforced by the evaluator.
package parser

import (
	"fmt"
	"strings"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/lexer"
	"github.com/npikall/toml-edit/token"
)

// ParseError is a syntax error at a source position.
type ParseError struct {
	Line int // 1-based
	Col  int // 1-based, counted in bytes
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg)
}

type parser struct {
	toks     []token.Token // always ends with EOF
	pos      int
	strict10 bool
}

// Option configures Parse.
type Option func(*parser)

// Strict10 makes Parse reject syntax that TOML 1.1 added to 1.0: the
// escapes \xHH and \e, times without seconds, and newlines, comments and a
// trailing comma inside inline tables.
func Strict10() Option {
	return func(p *parser) { p.strict10 = true }
}

// Parse parses src as TOML 1.1, or 1.0 with Strict10, into a Document whose
// String method returns src unchanged. It stops at the first syntax error.
func Parse(src string, opts ...Option) (*ast.Document, error) {
	p := &parser{}
	for _, opt := range opts {
		opt(p)
	}
	var lexOpts []lexer.Option
	if p.strict10 {
		lexOpts = append(lexOpts, lexer.Strict10())
	}
	l := lexer.New(src, lexOpts...)
	for {
		tok := l.NextToken()
		p.toks = append(p.toks, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	return p.parseDocument()
}

func (p *parser) parseDocument() (*ast.Document, error) {
	doc := &ast.Document{Root: &ast.Table{}}
	current := doc.Root
	for {
		leading := p.trivia(true)
		switch tok := p.peek(); {
		case tok.Type == token.EOF:
			doc.Trailing = leading
			return doc, nil
		case isSimpleKey(tok.Type):
			kv, err := p.parseKeyValue()
			if err != nil {
				return nil, err
			}
			kv.Leading = leading
			if kv.Trailing, err = p.lineEnd(); err != nil {
				return nil, err
			}
			current.Body = append(current.Body, kv)
		case tok.Type == token.LBRACKET, tok.Type == token.DOUBLE_LBRACKET:
			header, err := p.parseTableHeader()
			if err != nil {
				return nil, err
			}
			header.Leading = leading
			current = &ast.Table{Header: header}
			doc.Tables = append(doc.Tables, current)
		default:
			return nil, unexpected(tok, "a key or table header")
		}
	}
}

// parseTableHeader parses "[key]" or "[[key]]" up to the end of its line.
func (p *parser) parseTableHeader() (*ast.TableHeader, error) {
	open := p.next()
	header := &ast.TableHeader{ArrayOfTables: open.Type == token.DOUBLE_LBRACKET}
	closing, closingWant := token.TokenType(token.RBRACKET), `"]"`
	if header.ArrayOfTables {
		closing, closingWant = token.DOUBLE_RBRACKET, `"]]"`
	}
	prefix := p.trivia(false)
	key, err := p.parseKey()
	if err != nil {
		return nil, err
	}
	key.Parts[0].Prefix = prefix
	header.Key = key
	if err := p.expect(closing, closingWant); err != nil {
		return nil, err
	}
	if header.Trailing, err = p.lineEnd(); err != nil {
		return nil, err
	}
	return header, nil
}

func (p *parser) parseKeyValue() (*ast.KeyValue, error) {
	key, err := p.parseKey()
	if err != nil {
		return nil, err
	}
	if err := p.expect(token.EQUALS, `"="`); err != nil {
		return nil, err
	}
	prefix := p.trivia(false)
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	ast.ValueDecor(value).Prefix = prefix
	return &ast.KeyValue{Key: key, Value: value}, nil
}

// parseKey parses a simple or dotted key. Whitespace after each part, up to
// the next "." or the end of the key, is that part's suffix.
func (p *parser) parseKey() (*ast.Key, error) {
	key := &ast.Key{}
	prefix := ""
	for {
		tok := p.peek()
		if !isSimpleKey(tok.Type) {
			return nil, unexpected(tok, "a key")
		}
		p.next()
		part := &ast.KeyPart{Type: tok.Type, Raw: tok.Literal, Pos: pos(tok)}
		part.Prefix = prefix
		part.Suffix = p.trivia(false)
		key.Parts = append(key.Parts, part)
		if p.peek().Type != token.DOT {
			return key, nil
		}
		p.next()
		prefix = p.trivia(false)
	}
}

//nolint:ireturn // ast.Value is a closed sum of Scalar, Array and InlineTable.
func (p *parser) parseValue() (ast.Value, error) {
	switch tok := p.peek(); {
	case isScalar(tok.Type):
		p.next()
		return &ast.Scalar{Type: tok.Type, Raw: tok.Literal, Pos: pos(tok)}, nil
	case tok.Type == token.LBRACKET:
		return p.parseArray()
	case tok.Type == token.LBRACE:
		return p.parseInlineTable()
	default:
		return nil, unexpected(tok, "a value")
	}
}

// parseArray parses "[v1, v2, ...]". Comments and newlines may appear
// anywhere between the brackets.
func (p *parser) parseArray() (*ast.Array, error) {
	p.next() // [
	arr := &ast.Array{}
	for {
		prefix := p.trivia(true)
		if p.peek().Type == token.RBRACKET {
			p.next()
			arr.Trailing = prefix
			return arr, nil
		}
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		decor := ast.ValueDecor(value)
		decor.Prefix = prefix
		decor.Suffix = p.trivia(true)
		item := &ast.ArrayItem{Value: value}
		arr.Items = append(arr.Items, item)
		arr.TrailingComma = false
		switch tok := p.next(); tok.Type {
		case token.COMMA:
			arr.TrailingComma = true
			item.AfterComma = p.afterComma()
		case token.RBRACKET:
			return arr, nil
		default:
			return nil, unexpected(tok, `"," or "]"`)
		}
	}
}

// parseInlineTable parses "{k1 = v1, k2 = v2, ...}". As of TOML 1.1,
// comments, newlines and a trailing comma are allowed.
func (p *parser) parseInlineTable() (*ast.InlineTable, error) {
	p.next() // {
	table := &ast.InlineTable{}
	toml11 := !p.strict10 // allow newlines, comments and a trailing comma
	for {
		leading := p.trivia(toml11)
		if tok := p.peek(); tok.Type == token.RBRACE {
			if !toml11 && table.TrailingComma {
				return nil, unexpected(tok, "a key")
			}
			p.next()
			table.Trailing = leading
			return table, nil
		}
		kv, err := p.parseKeyValue()
		if err != nil {
			return nil, err
		}
		kv.Leading = leading
		kv.Trailing = p.trivia(toml11)
		entry := &ast.InlineEntry{KeyValue: kv}
		table.Entries = append(table.Entries, entry)
		table.TrailingComma = false
		switch tok := p.next(); tok.Type {
		case token.COMMA:
			table.TrailingComma = true
			if toml11 {
				entry.AfterComma = p.afterComma()
			}
		case token.RBRACE:
			return table, nil
		default:
			return nil, unexpected(tok, `"," or "}"`)
		}
	}
}

// lineEnd consumes optional whitespace and a comment, then the newline that
// ends a top-level expression. At EOF no newline is required.
func (p *parser) lineEnd() (string, error) {
	var sb strings.Builder
	sb.WriteString(p.trivia(false))
	if p.peek().Type == token.COMMENT {
		sb.WriteString(p.next().Literal)
	}
	switch tok := p.peek(); tok.Type {
	case token.NEWLINE:
		sb.WriteString(p.next().Literal)
	case token.EOF:
	default:
		return "", unexpected(tok, "a newline")
	}
	return sb.String(), nil
}

// afterComma consumes the rest of the line after a comma inside an array or
// inline table: whitespace, an optional comment and the newline. If the line
// does not end there, nothing is consumed and the whitespace stays with the
// next element.
func (p *parser) afterComma() string {
	start := p.pos
	var sb strings.Builder
	sb.WriteString(p.trivia(false))
	if p.peek().Type == token.COMMENT {
		sb.WriteString(p.next().Literal)
	}
	if p.peek().Type != token.NEWLINE {
		p.pos = start
		return ""
	}
	sb.WriteString(p.next().Literal)
	return sb.String()
}

// trivia consumes whitespace, and if multiline is set also comments and
// newlines, and returns their concatenated text.
func (p *parser) trivia(multiline bool) string {
	var sb strings.Builder
	for {
		switch p.peek().Type {
		case token.WHITESPACE:
		case token.COMMENT, token.NEWLINE:
			if !multiline {
				return sb.String()
			}
		default:
			return sb.String()
		}
		sb.WriteString(p.next().Literal)
	}
}

func (p *parser) expect(typ token.TokenType, what string) error {
	if tok := p.peek(); tok.Type != typ {
		return unexpected(tok, what)
	}
	p.next()
	return nil
}

func (p *parser) peek() token.Token {
	return p.toks[p.pos]
}

func (p *parser) next() token.Token {
	tok := p.toks[p.pos]
	if tok.Type != token.EOF {
		p.pos++
	}
	return tok
}

func pos(tok token.Token) ast.Pos {
	return ast.Pos{Line: tok.Line, Col: tok.Col}
}

func unexpected(tok token.Token, want string) *ParseError {
	var got string
	switch tok.Type {
	case token.EOF:
		got = "end of file"
	case token.NEWLINE:
		got = "newline"
	case token.ILLEGAL:
		got = fmt.Sprintf("invalid %q", tok.Literal)
	default:
		got = fmt.Sprintf("%q", tok.Literal)
	}
	return &ParseError{Line: tok.Line, Col: tok.Col, Msg: fmt.Sprintf("expected %s, found %s", want, got)}
}

func isSimpleKey(typ token.TokenType) bool {
	return typ == token.BARE_KEY || typ == token.BASIC_STRING || typ == token.LITERAL_STRING
}

func isScalar(typ token.TokenType) bool {
	switch typ {
	case token.BASIC_STRING, token.LITERAL_STRING, token.ML_BASIC_STRING, token.ML_LITERAL_STRING,
		token.INTEGER, token.FLOAT, token.BOOL,
		token.OFFSET_DATETIME, token.LOCAL_DATETIME, token.LOCAL_DATE, token.LOCAL_TIME:
		return true
	}
	return false
}
