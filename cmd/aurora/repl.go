package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/repl"
)

func init() {
	replCmd.Flags().StringSliceP("loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		loggers, err := cmd.Flags().GetStringSlice("loggers")
		if err != nil {
			return err
		}
		repl.Start(os.Stdin, loggers)
		return nil
	},
}
