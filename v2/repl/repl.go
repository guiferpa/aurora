package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := bytes.NewBufferString(scanner.Text())

		fmt.Println("--- TOKENS ---")
		l, err := lexer.GetFilledTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, tok := range l {
			fmt.Printf("Line: %d, Column: %d, Tag: %s, Match: %s\n", tok.GetLine(), tok.GetColumn(), tok.GetTag().Id, tok.GetMatch())
		}

		fmt.Println("--- AST ---")
		ast, err := parser.New(l).Parse()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(ast, "AAAAAAAAAAAAAAAAAAAAAAAAAAA")
	}
}
