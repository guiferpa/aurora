package lexer

type Lexer struct {
}

type NewLexerOptions struct {
}

func New(options NewLexerOptions) *Lexer {
	return &Lexer{}
}
