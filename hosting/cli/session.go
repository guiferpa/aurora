package cli

import (
	"errors"
	"io"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/loader"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ir"
)

// A Session is one use of the command line: the phases it was handed, how wide a value is,
// and where what it has to say goes. Every command is a method on it.
//
// The phases arrive built. Nothing here calls lexer.New, parser.New or emitter.New — that
// happens in cmd/aurora, once, next to the flags that decide how they are built, so there is
// one place to look for how the compiler was put together.
type Session struct {
	lexer   *lexer.Lexer
	parser  parser.Parser
	emitter emitter.Emitter
	// newEvaluator returns a fresh one. An evaluator is per program, not per session: it
	// holds the names a program bound, the scopes it deferred and the assertions that ran, and
	// "aurora test" checks one file after another, each of which is its own program.
	newEvaluator func() *evaluator.Evaluator
	// resolver finds the files a program is made of. It arrives built because what it
	// resolves from — the source root — is decided with the target, next to the flags.
	resolver *resolver.Resolver
	tapeSize int
	stdout   io.Writer
	warnings io.Writer
}

type NewSessionOptions struct {
	Lexer        *lexer.Lexer
	Parser       parser.Parser
	Emitter      emitter.Emitter
	NewEvaluator func() *evaluator.Evaluator
	Resolver     *resolver.Resolver
	// TapeSize is the width in bytes of every value. It is the width the phases were built
	// with; the session carries it to answer for what it wrote.
	TapeSize int
	// Stdout receives what a command has to say. Nil says nothing.
	Stdout io.Writer
	// Warnings receives compiler warnings. Nil discards them.
	Warnings io.Writer
}

// evaluator returns a fresh evaluator, or says that the session was built without a way
// of making one. A build needs none; a run and a test do, and printing nowhere by default
// would hide the wiring mistake in the one place it shows.
func (s *Session) evaluator() (*evaluator.Evaluator, error) {
	if s.newEvaluator == nil {
		return nil, errors.New("no evaluator was given to this session")
	}
	return s.newEvaluator(), nil
}

// compile turns the file somebody named into the program it is: every module it needs, in
// the order they load, with every qualified name checked and the whole thing in one stream.
//
// It is what every command that runs Aurora goes through. A file that imports nothing comes
// out of it as a program of one module, which is the shape everything already had.
// Blocks answers the program a source compiles to, for whoever needs to know what it does
// rather than to run it.
//
// It exists for the two commands that speak to a chain: what a scope does decides whether
// reaching it is a question or a change, and that is the compiler's to answer rather than
// somebody's to remember.
func (s *Session) Blocks(source string) ([]ir.Block, error) {
	program, err := s.compile(source)
	if err != nil {
		return nil, err
	}
	return program.Blocks, nil
}

func (s *Session) compile(source string) (loader.Program, error) {
	if s.resolver == nil {
		return loader.Program{}, errors.New("no resolver was given to this session")
	}
	modules, err := s.resolver.Resolve(source)
	if err != nil {
		return loader.Program{}, err
	}
	return loader.Load(modules, s.emitter.EmitProgram)
}

// report says what compiling each module had to say, naming the file it came from — a warning
// about a module is not about the file somebody asked to run.
func (s *Session) report(program loader.Program) {
	for _, each := range program.Ranges {
		ReportWarnings(s.warnings, each.Filename, each.Warnings)
	}
}

func NewSession(opts NewSessionOptions) *Session {
	return &Session{
		lexer:        opts.Lexer,
		parser:       opts.Parser,
		emitter:      opts.Emitter,
		newEvaluator: opts.NewEvaluator,
		resolver:     opts.Resolver,
		tapeSize:     opts.TapeSize,
		stdout:       opts.Stdout,
		warnings:     opts.Warnings,
	}
}
