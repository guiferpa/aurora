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

func isString(v []byte) bool {
	return len(v) > 8 && len(v)%8 == 0 && utf8.Valid(v)
}

func isBoolean(v []byte) bool {
	return len(v) == 1 && v[0] == 0
}

func isNothing(v []byte) bool {
	return byteutil.IsNothing(v)
}

func render(w io.Writer, temps map[string][]byte, eerr error) {
	marker := color.New(color.FgWhite, color.Bold).Sprint("=")
	literals := color.New(color.FgHiYellow).SprintFunc()
	internals := color.New(color.FgCyan).SprintFunc()
	errors := color.New(color.FgRed).SprintFunc()
	format := "%s %s\n"

	if eerr != nil {
		_, _ = fmt.Fprintf(w, format, marker, errors(eerr))
		return
	}

	for _, v := range temps {
		if isNothing(v) {
			_, _ = fmt.Fprintf(w, format, marker, internals("<nothing>"))
			continue
		}
		if isBoolean(v) {
			_, _ = fmt.Fprintf(w, format, marker, literals(byteutil.ToBoolean(v)))
			continue
		}
		if isString(v) {
			_, _ = fmt.Fprintf(w, format, marker, literals(string(v)))
			continue
		}
		er, err := byteutil.Encode(v)
		if err != nil {
			_, _ = fmt.Fprint(w, errors(err))
			break
		}
		_, _ = fmt.Fprintf(w, format, marker, internals(er))
	}
}

func Start(in io.Reader, loggers []string) {
	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(loggers, "evaluator"),
		EchoWriter:    &EchoWriter{},
		PrintWriter:   &PrintWriter{},
	})

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		fmt.Println("Bye :)")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(in)
	var instsBuffer []emitter.Instruction
	for {
		_, _ = fmt.Fprintf(os.Stdout, ">> ")
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

		// Append to buffer so defer from/to indices stay valid when calling later.
		from := uint64(len(instsBuffer))
		instsBuffer = append(instsBuffer, insts...)
		to := uint64(len(instsBuffer))

		temps, err := ev.EvaluateRange(instsBuffer, from, to)
		render(os.Stdout, temps, err)
		if err != nil {
			continue
		}
	}
}
