package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
//	"github.com/guiferpa/aurora/print" just for debugger
	"github.com/guiferpa/aurora/repl"
)

func run(args []string) {
	if len(args) != 1 {
		return
	}
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
	opcodes, err := emitter.New(ast).Emit()
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	ev := evaluator.New()
	if _, err := ev.Evaluate(opcodes); err != nil {
		color.Red("%v", err)
		os.Exit(5)
	}
	// print.Opcodes(os.Stdout, ev.GetOpCodes(), true) just for debugger
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return
	}
	cmd := args[0]
	if cmd == "repl" {
		repl.Start(os.Stdin, os.Stdout)
	}
	if cmd == "run" {
		run(args[1:])
	}
}
