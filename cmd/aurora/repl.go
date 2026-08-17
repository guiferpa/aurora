package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/hosting/repl"
)

func init() {
	replCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8)")
	replCmd.Flags().StringSliceP("loggers", "l", []string{}, "show what each phase produced (valid: lexer, parser, emitter)")
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		loggers, err := cmd.Flags().GetStringSlice("loggers")
		if err != nil {
			return err
		}
		tapeSize, err := cmd.Flags().GetInt("tape-size")
		if err != nil {
			return err
		}
		if err := cli.ValidateTapeSize(tapeSize); err != nil {
			return err
		}
		repl.Start(os.Stdin, os.Stdout, loggers, tapeSize)
		return nil
	},
}
