package lexer

type Lexer struct {
}

// New builds a lexer. It takes nothing: a lexer is the same for every source, and the source
// arrives at the read.
func New() *Lexer {
	return &Lexer{}
}
