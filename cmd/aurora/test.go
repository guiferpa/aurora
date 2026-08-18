package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/shared/printer"
)

var testCmd = &cobra.Command{
	Use:   "test [profile | file.test.ar]",
	Short: "Run the test files of a project",
	Long: `Run the test files of a project.

A test belongs to the source file of the same name: greeting.test.ar tests
greeting.ar, which runs first so the test can see what it declared.

With no argument, the "main" profile is used and its source directory is
searched, down to the leaves. A name selects another profile. A path runs that
file alone:

  aurora test                      the "main" profile
  aurora test dev                  the "dev" profile
  aurora test src/greeting.test.ar that file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

func init() {
	testCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8; overrides tape_size from aurora.toml)")
}

func runTest(cmd *cobra.Command, args []string) error {
	var target string
	if len(args) > 0 {
		target = args[0]
	}
	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	// Which files run, and how wide a value is in them, is settled before the phases are
	// built: a test file named directly belongs to a project, and the project decides the width.
	files, size, err := cli.TestFiles(target, tapeSize)
	if err != nil {
		return err
	}

	report, err := cli.NewSession(cli.NewSessionOptions{
		Lexer:   lexer.New(lexer.NewLexerOptions{}),
		Parser:  parser.New(parser.NewParserOptions{TapeSize: size}),
		Emitter: emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		NewEvaluator: func() *evaluator.Evaluator {
			return evaluator.New(evaluator.NewEvaluatorOptions{
				// A test says what held and what did not; what the program printed on the
				// way is not part of the report.
				PrintBytes:   printer.Bytes(io.Discard, size),
				PrintChars:   printer.Chars(io.Discard, size),
				PrintDecimal: printer.Decimal(io.Discard, size),
				TapeSize:     size,
				Asserts:      true,
			})
		},
		TapeSize: size,
		Stdout:   os.Stdout,
	}).Test(cmd.Context(), files)
	if err != nil {
		return err
	}
	if !report.OK() {
		// The report has already been written; this is only what the exit code carries, so
		// that a script or a CI job can tell what happened.
		if report.Failed > 0 {
			return fmt.Errorf("%d of %d assertions failed", report.Failed, report.Passed+report.Failed)
		}
		return fmt.Errorf("some test files could not run")
	}
	return nil
}
