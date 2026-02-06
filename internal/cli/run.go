package cli

import (
	"context"
	"io"
	"os"
	"slices"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/logger"
	"github.com/guiferpa/aurora/parser"
)

// RunInput is the input for the Run handler.
type RunInput struct {
	Source  string   // path to .ar source
	Loggers []string // enabled loggers
	Stdin   io.Reader
	Stdout  io.Writer // used for both Echo and Print
	Player  *evaluator.Player
}

// Run compiles and evaluates the Aurora source at Source.
func Run(ctx context.Context, in RunInput) error {
	bs, err := os.ReadFile(in.Source)
	if err != nil {
		return err
	}

	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: slices.Contains(in.Loggers, "lexer"),
	}).GetFilledTokens(bs)
	if err != nil {
		return err
	}

	ast, err := parser.New(tokens, parser.NewParserOptions{
		Filename:      in.Source,
		EnableLogging: slices.Contains(in.Loggers, "parser"),
	}).Parse()
	if err != nil {
		return err
	}

	insts, err := emitter.New(emitter.NewEmitterOptions{
		EnableLogging: slices.Contains(in.Loggers, "emitter"),
	}).Emit(ast)
	if err != nil {
		return err
	}

	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(in.Loggers, "evaluator"),
		EchoWriter:    in.Stdout,
		PrintWriter:   in.Stdout,
	})
	if in.Player != nil {
		ev.SetPlayer(in.Player)
	}
	if _, err := ev.Evaluate(insts); err != nil {
		return err
	}
	logger.AssertError(ev.GetAssertErrors(), in.Source)
	return nil
}
