package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func Start(in io.Reader, out io.Writer, debug bool) {
	ev := evaluator.New(debug)

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		fmt.Println("Bye :)")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, ">> ")
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

		insts, err := emitter.New().Emit(ast)
		if err != nil {
			fmt.Println(err)
			continue
		}

		emitter.Print(insts, debug)

		temps, err := ev.Evaluate(insts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		s := color.New(color.FgWhite, color.Bold).Sprint("=")
		for _, v := range temps {
			er, err := byteutil.Encode(v)
			if err != nil {
				color.Red("%v", err)
				break

			}
			fmt.Printf("%s %s\n", s, color.New(color.FgHiYellow).Sprintf("%v", er))
		}
	}
}
