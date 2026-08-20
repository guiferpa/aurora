package cli

import (
	"io"
	"os"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/shared/manifest"
	"github.com/guiferpa/aurora/shared/printer"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// A command is a method on a session, and a session is what cmd/aurora puts together. A test
// that asks what a command answers has to put it together the same way — which is also what
// keeps the assembly honest: if it stops making sense here, it stopped making sense there.

// sessionOpts is what a test cares about: where the output goes, how wide a value is, and what
// the program was applied to.
type sessionOpts struct {
	tapeSize int
	stdout   io.Writer
	warnings io.Writer
	args     []string
	// asserts turns assertions on and sends what a program prints nowhere, which is what
	// "aurora test" does: a test says what held, not what was printed on the way.
	asserts bool
}

// newTestResolver puts the front of the pipeline together the way cmd/aurora does: a test
// wires what main wires, since a host is handed its phases rather than building them.
func newTestResolver(tapeSize int) *resolver.Resolver {
	lx := lexer.New()
	ps := parser.New()

	return resolver.New(resolver.Options{
		SourceRoot: manifest.DefaultSourceRoot,
		Read:       os.ReadFile,
		Parse: func(filename string, id module.ID, source []byte, imports map[string]ast.Offer) (ast.AST, error) {
			tokens, err := lx.GetFilledTokens(source)
			if err != nil {
				return ast.AST{}, err
			}
			return ps.Parse(parser.ParseInput{
				Filename: filename,
				Tokens:   tokens,
				TapeSize: tapeSize,
				Module:   string(id),
				Imports:  imports,
			})
		},
		Header: func(source []byte) ([]ast.UseDeclaration, error) {
			tokens, err := lx.GetFilledTokens(source)
			if err != nil {
				return nil, err
			}
			return parser.ScanUses(tokens), nil
		},
	})
}

func newSession(t *testing.T, o sessionOpts) *Session {
	t.Helper()

	stdout := o.stdout
	if stdout == nil {
		stdout = io.Discard
	}
	printed := stdout
	if o.asserts {
		printed = io.Discard
	}
	size := o.tapeSize

	return NewSession(NewSessionOptions{
		Lexer:    lexer.New(),
		Parser:   parser.New(),
		Emitter:  emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		Resolver: newTestResolver(size),
		NewEvaluator: func() *evaluator.Evaluator {
			return evaluator.New(evaluator.NewEvaluatorOptions{
				PrintBytes:   printer.Bytes(printed, size),
				PrintChars:   printer.Chars(printed, size),
				PrintDecimal: printer.Decimal(printed, size),
				Args:         ParseArgs(o.args),
				TapeSize:     size,
				Asserts:      o.asserts,
			})
		},
		TapeSize: size,
		Stdout:   stdout,
		Warnings: o.warnings,
	})
}

// tested resolves a target the way "aurora test" does and runs what it found.
func tested(t *testing.T, target string, o sessionOpts) (TestReport, error) {
	t.Helper()

	files, size, err := TestFiles(target, o.tapeSize)
	if err != nil {
		return TestReport{}, err
	}
	o.tapeSize = size
	o.asserts = true

	return newSession(t, o).Test(t.Context(), files)
}
