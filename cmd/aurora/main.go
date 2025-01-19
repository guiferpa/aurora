package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/builder"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/logger"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/repl"
	"github.com/guiferpa/aurora/version"
)

var (
	player bool
	debug  bool
	output string
)

var evalCmd = &cobra.Command{
	Use:   "eval [file]",
	Short: "Evaluate aurora binary file built by build command",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bs := logger.MustError(os.ReadFile(args[0]))
		insts := logger.MustError(emitter.Parse(bs))
		emitter.Print(insts, debug)
		logger.MustError(evaluator.New(debug).Evaluate(insts))
	},
}

var buildCmd = &cobra.Command{
	Use:   "build [file]",
	Short: "Build binary from source code",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bs := logger.MustError(os.ReadFile(args[0]))
		tokens := logger.MustError(lexer.GetFilledTokens(bs))
		ast := logger.MustError(parser.New(tokens).Parse())
		insts := logger.MustError(emitter.New().Emit(ast))
		fd := os.Stdout
		if strings.Compare(output, "") != 0 {
			fd = logger.MustError(os.Create(output))
		}
		logger.MustError(builder.New(insts).Build(fd))
	},
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(os.Stdin, os.Stdout, debug)
	},
}

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Run program directly from source code",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bs := logger.MustError(os.ReadFile(args[0]))
		tokens := logger.MustError(lexer.GetFilledTokens(bs))
		ast := logger.MustError(parser.New(tokens).Parse())
		insts := logger.MustError(emitter.New().Emit(ast))
		emitter.Print(insts, debug)
		ev := evaluator.New(debug)
		if player && debug {
			ev = evaluator.NewWithPlayer(true, evaluator.NewPlayer(os.Stdin))
		}
		logger.MustError(ev.Evaluate(insts))
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show toolbox version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.VERSION)
	},
}

var rootCmd = &cobra.Command{
	Use: "aurora",
}

func main() {
	runCmd.Flags().BoolVarP(&player, "player", "p", false, "enable player for evaluator phase")
	runCmd.Flags().BoolVarP(&debug, "debug", "b", false, "enable debug for show deep dive logs from all phases")

	replCmd.Flags().BoolVarP(&debug, "debug", "b", false, "enable debug for show deep dive logs from all phases")

	buildCmd.Flags().StringVarP(&output, "output", "o", "", "set an output filename")
	evalCmd.Flags().BoolVarP(&debug, "debug", "b", false, "enable debug for show deep dive logs from all phases")

	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, evalCmd)
	logger.CommandError(rootCmd.Execute())
}
