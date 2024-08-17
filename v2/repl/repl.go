package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/fatih/color"
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
			env := ev.GetEnvironPool().Current()
			env.Print(os.Stdout)
			continue
		}

		if strings.Compare(scanner.Text(), "get_opcodes") == 0 {
			print.Opcodes(os.Stdout, ev.GetOpCodes(), false)
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
		s := color.New(color.FgWhite, color.Bold).Sprint("=")
		for _, v := range labels {
			fmt.Printf("%s %s\n", s, color.New(color.FgHiYellow).Sprintf("%d", v))
		}
	}
}
