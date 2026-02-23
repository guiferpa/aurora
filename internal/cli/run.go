package cli

import (
	"context"
	"io"
	"slices"

	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/linker"
	"github.com/guiferpa/aurora/logger"
)

// RunInput is the input for the Run handler.
type RunInput struct {
	Source  string   // path to .ar source
	Loggers []string // enabled loggers
	Stdin   io.Reader
	Stdout  io.Writer // used for both Echo and Print
	Player  *evaluator.Player
	Args    []string
}

// Run compiles and evaluates the Aurora source at Source.
func Run(ctx context.Context, in RunInput) error {
	l, err := linker.NewLinker(linker.NewLinkerOptions{
		Source:  in.Source,
		Loggers: in.Loggers,
	})
	if err != nil {
		return err
	}
	insts, err := l.Resolve()
	if err != nil {
		return err
	}

	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(in.Loggers, "evaluator"),
		EchoWriter:    in.Stdout,
		PrintWriter:   in.Stdout,
		Args:          ParseArgs(in.Args),
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
