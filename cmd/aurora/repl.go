package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/hosting/repl"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/shared/printer"
)

func init() {
	replCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8)")
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		tapeSize, err := cmd.Flags().GetInt("tape-size")
		if err != nil {
			return err
		}
		if err := byteutil.ValidateTapeSize(tapeSize); err != nil {
			return err
		}
		size := byteutil.TapeSize(tapeSize)
		out := os.Stdout

		repl.NewSession(repl.NewSessionOptions{
			Lexer:   lexer.New(lexer.NewLexerOptions{}),
			Parser:  parser.New(parser.NewParserOptions{}),
			Emitter: emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
			// One evaluator lasts the session: a name bound on one line is there on the next.
			Evaluator: evaluator.New(evaluator.NewEvaluatorOptions{
				PrintBytes:   printer.Bytes(out, size),
				PrintChars:   printer.Chars(out, size),
				PrintDecimal: printer.Decimal(out, size),
				TapeSize:     size,
			}),
			In:       os.Stdin,
			Out:      out,
			TapeSize: size,
		}).Start()
		return nil
	},
}
