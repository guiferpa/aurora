package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/builder"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/repl"
)

var (
	player bool
	debug  bool
	output string
)

var buildCmd = &cobra.Command{
	Use:  "build",
	Args: cobra.MinimumNArgs(1),
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
		fd := os.Stdout
		if strings.Compare(output, "") != 0 {
			file, err := os.Create(output)
			if err != nil {
				fmt.Println(err)
				os.Exit(5)
			}
			fd = file
		}
		if _, err := builder.New(insts).Build(fd); err != nil {
			fmt.Println(err)
			os.Exit(6)
		}
	},
}

var replCmd = &cobra.Command{
	Use: "repl",
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(os.Stdin, os.Stdout, debug)
	},
}

var runCmd = &cobra.Command{
	Use:  "run",
	Args: cobra.MinimumNArgs(1),
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
		emitter.Print(os.Stdout, insts, debug)
		ev := evaluator.New(debug)
		if player && debug {
			ev = evaluator.NewWithPlayer(true, evaluator.NewPlayer(os.Stdin))
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
	runCmd.Flags().BoolVarP(&player, "player", "p", false, "enable player for evaluator phase")
	runCmd.Flags().BoolVarP(&debug, "debug", "b", false, "enable debug for show deep dive logs from all phases")

	replCmd.Flags().BoolVarP(&debug, "debug", "b", false, "enable debug for show deep dive logs from all phases")

	buildCmd.Flags().StringVarP(&output, "output", "o", "", "set an output filename")

	rootCmd.AddCommand(runCmd, replCmd, buildCmd)
	if err := rootCmd.Execute(); err != nil {
		color.Red("%v", err)
		os.Exit(6)
	}
}
