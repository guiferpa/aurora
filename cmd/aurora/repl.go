package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/repl"
)

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(os.Stdin, debug, raw, loggers)
	},
}
