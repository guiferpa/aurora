package textdoc

import (
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// A Session is one editor session: the phases it was handed, and the questions an editor asks
// about a document. Every request the server answers is a method on it.
//
// The phases arrive built, from cmd/aurorals — the server is a host, and a host does not put
// the compiler together. What it does decide is the width a document is read at, which comes
// from the project the file belongs to and travels with each Document, since one session
// answers for files of any project the editor opens.
type Session struct {
	lexer   *lexer.Lexer
	parser  parser.Parser
	resolve Resolve
}

// Resolve answers with the modules a document imports, given the use lines it opens with.
//
// It takes the declarations rather than a tree because a document being edited does not
// parse, and what is inside a module is wanted exactly then — the moment a dot is typed.
// They are the top of the file and readable from the tokens alone.
//
// It is a port because finding those files is the host's business and each host does it
// differently: the server reads the editor's buffers and falls back to the disk, and the
// playground has no file system at all — it passes nothing, and a document there imports
// nothing, which is the truth about a page with one editor in it.
type Resolve func(doc Document, uses []ast.UseDeclaration) ([]module.Module, error)

type NewSessionOptions struct {
	Lexer  *lexer.Lexer
	Parser parser.Parser
	// Resolve is optional. Without it a document is analysed on its own, which is what it
	// was before modules existed.
	Resolve Resolve
}

func NewSession(opts NewSessionOptions) *Session {
	return &Session{lexer: opts.Lexer, parser: opts.Parser, resolve: opts.Resolve}
}
