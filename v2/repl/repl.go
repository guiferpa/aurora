package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/print"
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
		tokens, err := lexer.GetFilledTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		print.JSON(os.Stdout, tokens)

		fmt.Println("--- AST ---")
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			fmt.Println(err)
			continue
		}
		print.JSON(os.Stdout, ast)

		fmt.Println("--- Intermediate code ---")
		opcodes := emitter.NewThree(ast).Emit()
		print.JSON(os.Stdout, fmt.Sprintf("%s", opcodes))
	}
}
