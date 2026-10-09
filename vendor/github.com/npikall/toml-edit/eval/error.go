package eval

import (
	"fmt"

	"github.com/npikall/toml-edit/ast"
)

// Error is a semantic error at a source position.
type Error struct {
	Line int // 1-based
	Col  int // 1-based, counted in bytes
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg)
}

func errorAt(pos ast.Pos, format string, args ...any) *Error {
	return &Error{Line: pos.Line, Col: pos.Col, Msg: fmt.Sprintf(format, args...)}
}
