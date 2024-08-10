package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/guiferpa/aurora/lexer"
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
		l, err := lexer.GetTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(l)
	}
}
