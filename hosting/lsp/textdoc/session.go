package textdoc

import (
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// A Session is one editor session: the phases it was handed, and the questions an editor asks
// about a document. Every request the server answers is a method on it.
//
// The phases arrive built, from cmd/aurorals — the server is a host, and a host does not put
// the compiler together. What it does decide is the width a document is read at, which comes
// from the project the file belongs to and travels with each Document, since one session
// answers for files of any project the editor opens.
type Session struct {
	lexer  *lexer.Lexer
	parser parser.Parser
}

type NewSessionOptions struct {
	Lexer  *lexer.Lexer
	Parser parser.Parser
}

func NewSession(opts NewSessionOptions) *Session {
	return &Session{lexer: opts.Lexer, parser: opts.Parser}
}
