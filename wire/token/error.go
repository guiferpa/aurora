package token

import "fmt"

// Error is a compiler error that carries the position of the offending input, so callers
// can point at it without parsing the message. The CLI prints Message as before; the
// language server turns Offset and Length into a range.
//
// Line and Column are 1-based and Column counts bytes, matching the token positions.
// Offset is the byte offset of the token in the source, which is what position mapping
// should use: it survives multi-byte runes, where a byte column does not.
type Error struct {
	Message string
	Line    int
	Column  int
	Offset  int
	Length  int
}

func (e *Error) Error() string {
	return e.Message
}

// NewError builds an Error positioned at tok. The message is formatted by the caller so
// existing wording stays untouched.
func NewError(tok Token, format string, args ...any) *Error {
	err := &Error{Message: fmt.Sprintf(format, args...), Length: 1}
	if tok == nil {
		return err
	}
	err.Line = tok.GetLine()
	err.Column = tok.GetColumn()
	err.Offset = tok.GetCursor()
	if length := len(tok.GetMatch()); length > 0 {
		err.Length = length
	}
	return err
}
