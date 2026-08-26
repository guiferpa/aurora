package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/shared/printer"
)

var runCmd = &cobra.Command{
	Use:   "run [profile | file.ar] [args...]",
	Short: "Run program directly from source code",
	Long: `Run program directly from source code.

With no argument, the "main" profile from aurora.toml is used. A name selects
another profile. A path ending in .ar runs that file, with no manifest
involved:

  aurora run                  the "main" profile
  aurora run dev              the "dev" profile
  aurora run examples/x.ar    that file

Anything after the target is passed to the program and read with feed(n).`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8; overrides tape_size from aurora.toml)")
	runCmd.Flags().String("value", "0", "wei the program is told the transaction carried, which is what callvalue reads")
}

func runRun(cmd *cobra.Command, args []string) error {
	written, err := cmd.Flags().GetString("value")
	if err != nil {
		return err
	}
	carried, err := cli.ParseCallValue(written)
	if err != nil {
		return err
	}

	var arg string
	var programArgs []string
	if len(args) > 0 {
		arg, programArgs = args[0], args[1:]
	}
	target, err := cli.ResolveTarget(arg)
	if err != nil {
		return err
	}

	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	size := cli.ResolveTapeSize(tapeSize, target.TapeSize)
	out := os.Stdout

	return cli.NewSession(cli.NewSessionOptions{
		Lexer:    lexer.New(),
		Parser:   parser.New(),
		Emitter:  emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		Resolver: newResolver(size, target.SourceRoot),
		NewEvaluator: func() *evaluator.Evaluator {
			return evaluator.New(evaluator.NewEvaluatorOptions{
				PrintBytes:   printer.Bytes(out, size),
				PrintChars:   printer.Chars(out, size),
				PrintDecimal: printer.Decimal(out, size),
				Args:         cli.ParseArgs(programArgs),
				// What the transaction carried, off a chain: whoever runs the program says
				// so, the way they say what is applied to it. Without that, a program that
				// reads it could not be simulated at all — and the whole point of this
				// backend is that the same program answers the same thing either way.
				CallValue: carried,
				TapeSize:  size,
			})
		},
		TapeSize: size,
		Stdout:   out,
		Warnings: os.Stderr,
	}).Run(cmd.Context(), target.Source)
}
