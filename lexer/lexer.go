package lexer

type Lexer struct {
	enableLogging bool
}

type NewLexerOptions struct {
	EnableLogging bool
}

func New(options NewLexerOptions) *Lexer {
	return &Lexer{enableLogging: options.EnableLogging}
}
