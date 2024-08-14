package repl

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/print"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	ev := evaluator.New()

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		fmt.Println("Bye :)")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		if strings.Compare(scanner.Text(), "get_memory") == 0 {
			for k, v := range ev.GetMemory() {
				fmt.Printf("%s: %x\n", k, v)
			}
			continue
		}

		if strings.Compare(scanner.Text(), "get_opcodes") == 0 {
			print.Opcodes(os.Stdout, ev.GetOpCodes())
			continue
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

		opcodes, err := emitter.New(ast).Emit()
		if err != nil {
			fmt.Println(err)
			continue
		}

		labels, err := ev.Evaluate(opcodes)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, v := range labels {
			d := binary.BigEndian.Uint64(v)
			fmt.Printf("= %d\n", d)
		}
	}
}
