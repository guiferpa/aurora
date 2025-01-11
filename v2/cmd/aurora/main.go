package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/repl"
)

var withPlayer bool

var replCmd = &cobra.Command{
	Use: "repl",
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(os.Stdin, os.Stdout)
	},
}

var runCmd = &cobra.Command{
	Use: "run",
	Run: func(cmd *cobra.Command, args []string) {
		bs, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		tokens, err := lexer.GetFilledTokens(bs)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}
		insts, err := emitter.New(ast).Emit()
		if err != nil {
			fmt.Println(err)
			os.Exit(4)
		}
		emitter.Print(os.Stdout, insts)
		ev := evaluator.New()
		if withPlayer {
			ev = evaluator.NewWithPlayer(evaluator.NewPlayer(os.Stdin))
		}
		if _, err := ev.Evaluate(insts); err != nil {
			color.Red("%v", err)
			os.Exit(5)
		}
	},
}

var rootCmd = &cobra.Command{
	Use: "aurora",
}

func run(args []string) {
	if len(args) != 1 {
		return
	}
}

func main() {
	runCmd.Flags().BoolVarP(&withPlayer, "with-player", "w", false, "enable player for evaluator phase")

	rootCmd.AddCommand(runCmd, replCmd)
	if err := rootCmd.Execute(); err != nil {
		color.Red("%v", err)
		os.Exit(6)
	}
}
