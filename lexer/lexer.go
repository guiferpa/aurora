package lexer

type Lexer struct {
	logger *Logger
}

type NewLexerOptions struct {
	EnableLogging bool
}

func New(options NewLexerOptions) *Lexer {
	return &Lexer{logger: NewLogger(options.EnableLogging)}
}
