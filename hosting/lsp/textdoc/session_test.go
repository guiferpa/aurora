package textdoc

import (
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// session builds one the way cmd/aurorals does: the phases come from outside, and a test that
// asks what the editor is told has to hand them over the same way.
func session() *Session {
	return NewSession(NewSessionOptions{Lexer: lexer.New(), Parser: parser.New()})
}
