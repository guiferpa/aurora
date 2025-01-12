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
	"github.com/guiferpa/aurora/version"
)

var (
	player bool
	debug  bool
	output string
)

var evalCmd = &cobra.Command{
	Use:  "eval",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bs, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		insts, err := emitter.Parse(bs)
		if err != nil {
			return err
		}
		emitter.Print(insts, debug)
		if _, err = evaluator.New(debug).Evaluate(insts); err != nil {
			color.New(color.BgBlack, color.FgRed).Println(err)
			os.Exit(1)
		}
		return err
	},
}

var buildCmd = &cobra.Command{
	Use:  "build",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bs, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		tokens, err := lexer.GetFilledTokens(bs)
		if err != nil {
			return err
		}
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			return err
		}
		insts, err := emitter.New(ast).Emit()
		if err != nil {
			return err
		}
		fd := os.Stdout
		if strings.Compare(output, "") != 0 {
			file, err := os.Create(output)
			if err != nil {
				return err
			}
			fd = file
		}
		if _, err := builder.New(insts).Build(fd); err != nil {
			return err
		}
		return nil
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
	RunE: func(cmd *cobra.Command, args []string) error {
		bs, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		tokens, err := lexer.GetFilledTokens(bs)
		if err != nil {
			return err
		}
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			return err
		}
		insts, err := emitter.New(ast).Emit()
		if err != nil {
			return err
		}
		emitter.Print(insts, debug)
		ev := evaluator.New(debug)
		if player && debug {
			ev = evaluator.NewWithPlayer(true, evaluator.NewPlayer(os.Stdin))
		}
		if _, err := ev.Evaluate(insts); err != nil {
			color.New(color.BgBlack, color.FgRed).Println(err)
			os.Exit(1)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use: "version",
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
	if err := rootCmd.Execute(); err != nil {
		color.Red("%v", err)
		os.Exit(1)
	}
}
