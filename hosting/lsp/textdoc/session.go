package textdoc

import (
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/diag"
	"github.com/guiferpa/aurora/wire/ir"
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
	emit    Emit
	carries Carries
}

// Emit compiles a tree, which is how the editor hears what the compiler has to say about a
// document beyond whether it parses.
//
// A tree that parses can still be worth a word — a call applying fewer values than the scope
// it reaches reads, a scope holding more deferred scopes than a tape can name — and until
// this the editor heard none of it: analysis stopped at the parser, and everything the
// emitter says was said only to whoever ran "aurora build".
//
// It is a port for the same reason Resolve is: a host does not put the compiler together.
type Emit func(ast.AST) (ir.Program, error)

// Carries answers what a backend cannot carry of a compiled program.
//
// It is a second port because it is a second question, asked of a different phase. The emitter
// says what is wrong with a program; a backend says what is missing from the binary it would
// write — a feature it does not compile yet, and a print, whose log has nowhere to go on a
// chain.
//
// The editor is where that is cheapest to hear. Before this it was only said by "aurora
// build", which is after the writing is done and often after the deciding is done too.
type Carries func([]ir.Block) []diag.Warning

// Resolve returns the modules a document imports, given the use lines it opens with.
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
	// Emit is optional. Without it a document is only checked as far as it parses.
	Emit Emit
	// Carries is optional too, and needs Emit: it is asked of what Emit answered. Without it
	// a document hears nothing about the binary it would compile to.
	Carries Carries
}

func NewSession(opts NewSessionOptions) *Session {
	return &Session{lexer: opts.Lexer, parser: opts.Parser, resolve: opts.Resolve, emit: opts.Emit, carries: opts.Carries}
}
