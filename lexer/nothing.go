package lexer

func NewNothingToken(x, y, c int) Token {
	return &tok{
		x:     x,
		y:     y,
		c:     c,
		tag:   TagNothing,
		match: []byte("nothing"),
	}
}
