package cli

import (
	"context"
	"io"
	"slices"

	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/logger"
)

// RunInput is the input for the Run handler.
type RunInput struct {
	Source  string   // path to .ar source
	Loggers []string // enabled loggers
	Stdin   io.Reader
	Stdout  io.Writer // print: raw bytes
	// EchoOut receives echo output, which is text rather than bytes. When nil, Stdout
	// takes both, which is what a caller with a single stream wants.
	EchoOut io.Writer
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
	program, err := Compile(in.Source, in.TapeSize, in.Loggers)
	if err != nil {
		return err
	}
	ReportWarnings(in.Warnings, in.Source, program.Warnings)

	echoOut := in.EchoOut
	if echoOut == nil {
		echoOut = in.Stdout
	}
	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(in.Loggers, "evaluator"),
		EchoWriter:    echoOut,
		PrintWriter:   in.Stdout,
		Args:          ParseArgs(in.Args),
		TapeSize:      in.TapeSize,
	})
	if in.Player != nil {
		ev.SetPlayer(in.Player)
	}
	if _, err := ev.Evaluate(program.Instructions); err != nil {
		return err
	}
	logger.AssertError(ev.GetAssertErrors(), in.Source)
	return nil
}
