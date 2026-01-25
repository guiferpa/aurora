package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func printReadable(w io.Writer, temps map[string][]byte) {
	s := color.New(color.FgWhite, color.Bold).Sprint("=")
	for _, v := range temps {
		isBoolean := len(v) == 1 && v[0] == 0
		if isBoolean {
			_, _ = fmt.Fprintf(w, "%s %s\n", s, color.New(color.FgHiYellow).Sprint(byteutil.ToBoolean(v)))
			continue
		}
		isString := len(v) > 8 && len(v)%8 == 0
		if isString && utf8.Valid(v) {
			_, _ = fmt.Fprintf(w, "%s %s\n", s, color.New(color.FgHiYellow).Sprint(string(v)))
			continue
		}
		er, err := byteutil.Encode(v)
		if err != nil {
			_, _ = fmt.Fprint(w, color.New(color.FgRed).Sprint(err))
			break
		}
		_, _ = fmt.Fprintf(w, "%s %s\n", s, color.New(color.FgHiYellow).Sprint(er))
	}
}

func printRaw(w io.Writer, temps map[string][]byte) {
	s := color.New(color.FgWhite, color.Bold).Sprint("=")
	for _, v := range temps {
		_, _ = fmt.Fprintf(w, "%s %v\n", s, color.New(color.FgHiMagenta).Sprint(v))
	}
}

func Start(in io.Reader, out io.Writer, debug bool, raw bool, loggers []string) {
	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(loggers, "evaluator"),
		EchoWriter:    out,
		PrintWriter:   out,
	})

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		fmt.Println("Bye :)")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(in)
	for {
		_, _ = fmt.Fprintf(out, ">> ")
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := bytes.NewBufferString(scanner.Text())

		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: slices.Contains(loggers, "lexer"),
		}).GetFilledTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}

		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      "",
			EnableLogging: slices.Contains(loggers, "parser"),
		}).Parse()
		if err != nil {
			fmt.Println(err)
			continue
		}

		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: slices.Contains(loggers, "emitter"),
		}).Emit(ast)
		if err != nil {
			fmt.Println(err)
			continue
		}

		temps, err := ev.Evaluate(insts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if raw {
			printRaw(out, temps)
			continue
		}
		printReadable(out, temps)
	}
}
