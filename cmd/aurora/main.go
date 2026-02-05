package main

import (
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
	"github.com/guiferpa/aurora/logger"
)

var (
	player bool
	debug  bool
	raw    bool
	output string
)

var loggers []string

var rootCmd = &cobra.Command{
	Use: "aurora",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		switch args[0] {
		case "init", "version", "help", "repl":
			return nil
		}
		return cli.RequireManifest()
	},
}

func main() {
	runCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	runCmd.Flags().BoolVarP(&player, "player", "p", false, "enable player mode (stdin)")
	replCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	replCmd.Flags().BoolVarP(&raw, "raw", "r", false, "enable raw mode for show raw output")
	buildCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), builder)")
	buildCmd.Flags().StringVarP(&output, "output", "o", "", "output path for compiled binary (default: target from manifest)")

	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, deployCmd, callCmd, initCmd)

	logger.CommandError(rootCmd.Execute())
}
