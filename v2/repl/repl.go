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
	"github.com/guiferpa/aurora/runner"
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

		tokens, err := lexer.GetFilledTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}

		ast, err := parser.New(tokens).Parse()
		if err != nil {
			fmt.Println(err)
			continue
		}

		opcodes := emitter.New(ast).Emit()

		runner.New(opcodes).Run()

		print.JSON(os.Stdout, ast)
		print.Opcodes(os.Stdout, opcodes)
	}
}
