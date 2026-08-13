package repl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strings"
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
		_, _ = fmt.Fprintf(w, format, marker, literals(er))
	}
}

const prompt = ">> "

// lineReader is where the REPL gets the next line from: the editor when stdin is a
// terminal (arrow keys, history), a plain scanner otherwise (pipes, CI, tests).
type lineReader interface {
	ReadLine() (string, error)
}

// scannerReader is the non-interactive path: same behavior the REPL had before the editor.
type scannerReader struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func (s *scannerReader) ReadLine() (string, error) {
	_, _ = fmt.Fprint(s.out, prompt)
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return s.scanner.Text(), nil
}

// newLineReader picks the editor for a terminal and the scanner for anything else.
// The history is shared with the caller so it can record accepted lines.
func newLineReader(in io.Reader) (lineReader, *History) {
	f, ok := in.(*os.File)
	if !ok || !isTTY(f) {
		return &scannerReader{scanner: bufio.NewScanner(in), out: os.Stdout}, LoadHistory("")
	}

	// Without a home directory the history stays in memory; that is no reason to break the REPL.
	path, err := DefaultHistoryPath()
	if err != nil {
		path = ""
	}
	hist := LoadHistory(path)

	return newEditor(f, os.Stdout, prompt, hist, func() (func(), error) {
		return enterRaw(f)
	}), hist
}

func Start(in io.Reader, loggers []string) {
	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		EnableLogging: slices.Contains(loggers, "evaluator"),
		EchoWriter:    &EchoWriter{},
		PrintWriter:   &PrintWriter{},
	})

	reader, hist := newLineReader(in)
	editing, interactive := reader.(*editor)

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		// os.Exit skips defers, so leave raw mode here before quitting.
		if interactive {
			editing.Restore()
		}
		fmt.Println("Bye :)")
		os.Exit(0)
	}()

	var instsBuffer []emitter.Instruction
	histWarned := false
	for {
		text, err := reader.ReadLine()
		if errors.Is(err, errInterrupt) { // Ctrl+C: drop the line, prompt again
			continue
		}
		if err != nil { // Ctrl+D or end of piped input
			if interactive {
				fmt.Println("Bye :)")
			}
			return
		}
		if strings.TrimSpace(text) == "" {
			continue
		}

		// Record before evaluating so a line that fails to compile is still recallable.
		if err := hist.Append(text); err != nil && !histWarned {
			histWarned = true
			_, _ = fmt.Fprintf(os.Stderr, "warning: could not write history: %v\n", err)
		}

		line := bytes.NewBufferString(text)

		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: slices.Contains(loggers, "lexer"),
		}).GetFilledTokens(line.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}

		ast, err := parser.New(parser.NewParserOptions{
			Units: []parser.ParserUnit{
				{
					Filename:  "",
					Namespace: "main",
					Tokens:    tokens,
				},
			},
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
