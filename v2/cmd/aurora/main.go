package main

import (
	"fmt"
	"os"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/repl"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		repl.Start(os.Stdin, os.Stdout)
		return
	}
	bs, err := os.ReadFile(args[0])
	if err != nil {
		panic(err)
	}
	tokens, err := lexer.GetFilledTokens(bs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ast, err := parser.New(tokens).Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println(ast)
}
