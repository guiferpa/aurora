package cli

import (
	"context"
	"io"

	"github.com/guiferpa/aurora/evaluator"
)

// RunInput is the input for the Run handler.
type RunInput struct {
	Source  string   // path to .ar source
	Loggers []string // enabled loggers
	Stdin   io.Reader
	// Stdout receives what the program prints. The three print builtins are three
	// readings of the same tape and share one stream, so there is one writer here.
	Stdout io.Writer
	// Warnings receives compiler warnings. Nil discards them.
	Warnings io.Writer
	Player   *evaluator.Player
	Args     []string
	TapeSize int // width in bytes of every value; zero means the default
}

// Run compiles and evaluates the Aurora source at Source.
func Run(ctx context.Context, in RunInput) error {
	if err := ValidateTapeSize(in.TapeSize); err != nil {
		return err
	}
	program, err := Compile(in.Source, in.TapeSize, in.Loggers, in.Stdout)
	if err != nil {
		return err
	}
	ReportWarnings(in.Warnings, in.Source, CompilerWarnings(program.Warnings))

	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		Output:   in.Stdout,
		Args:     ParseArgs(in.Args),
		TapeSize: in.TapeSize,
	})
	if in.Player != nil {
		ev.SetPlayer(in.Player)
	}
	if _, err := ev.Evaluate(program.Instructions); err != nil {
		return err
	}
	return nil
}
