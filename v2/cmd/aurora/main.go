package main

import (
	"fmt"
	"os"

	"github.com/guiferpa/aurora/lexer"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		return
	}
	bs, err := os.ReadFile(args[0])
	if err != nil {
		panic(err)
	}
	tokens, err := lexer.GetTokensGivenBytes(bs)
	if err != nil {
		panic(err)
	}
	size := 0
	for _, t := range tokens {
		size += len(t.GetMatch())
		fmt.Printf("Line: %d, Column: %d, Tag: %s, Match: %v\n", t.GetLine(), t.GetColumn(), t.GetTag().Id, t.GetMatch())
	}
	fmt.Println("----------------------")
	fmt.Printf("Size: %d bytes\n", size)
}
