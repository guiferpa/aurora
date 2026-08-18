package cli

import (
	"errors"
	"io"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
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
	// newEvaluator answers with a fresh one. An evaluator is per program, not per session: it
	// holds the names a program bound, the scopes it deferred and the assertions that ran, and
	// "aurora test" checks one file after another, each of which is its own program.
	newEvaluator func() *evaluator.Evaluator
	tapeSize     int
	loggers      []string
	stdout       io.Writer
	warnings     io.Writer
}

type NewSessionOptions struct {
	Lexer        *lexer.Lexer
	Parser       parser.Parser
	Emitter      emitter.Emitter
	NewEvaluator func() *evaluator.Evaluator
	// TapeSize is the width in bytes of every value. It is the width the phases were built
	// with; the session carries it to answer for what it wrote.
	TapeSize int
	// Loggers names the phases whose output is shown: lexer, parser, emitter.
	Loggers []string
	// Stdout receives what a command has to say. Nil says nothing.
	Stdout io.Writer
	// Warnings receives compiler warnings. Nil discards them.
	Warnings io.Writer
}

// evaluator answers with a fresh evaluator, or says that the session was built without a way
// of making one. A build needs none; a run and a test do, and printing nowhere by default
// would hide the wiring mistake in the one place it shows.
func (s *Session) evaluator() (*evaluator.Evaluator, error) {
	if s.newEvaluator == nil {
		return nil, errors.New("no evaluator was given to this session")
	}
	return s.newEvaluator(), nil
}

func NewSession(opts NewSessionOptions) *Session {
	return &Session{
		lexer:        opts.Lexer,
		parser:       opts.Parser,
		emitter:      opts.Emitter,
		newEvaluator: opts.NewEvaluator,
		tapeSize:     opts.TapeSize,
		loggers:      opts.Loggers,
		stdout:       opts.Stdout,
		warnings:     opts.Warnings,
	}
}
