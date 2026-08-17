package token

// A Token is one thing the lexer recognised: what it matched, what kind of thing it is, and
// where it was written. It crosses from the lexer into the tree the parser builds and out to
// a host that wants to underline it, so it belongs to none of the three.
type Token interface {
	GetMatch() []byte
	GetTag() Tag
	GetLine() int
	GetColumn() int
	GetCursor() int
}

type tok struct {
	x, y, c int
	tag     Tag
	match   []byte
}

func (t tok) GetMatch() []byte {
	return t.match
}

func (t tok) GetTag() Tag {
	return t.tag
}

func (t tok) GetLine() int {
	return t.x
}

func (t tok) GetColumn() int {
	return t.y
}

func (t tok) GetCursor() int {
	return t.c
}

// New makes a token. The lexer is what calls it, but the shape is not the lexer's: a test, a
// language server, anything that has to speak in tokens builds one the same way.
func New(match []byte, tag Tag, line, column, cursor int) Token {
	return tok{line, column, cursor, tag, match}
}
