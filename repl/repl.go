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

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/internal/trace"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// render prints the value of the line that was typed — the temp left by its last
// instruction. Printing the whole temp map would spill every intermediate value of the
// expression, in map order, which is no order at all.
func render(w io.Writer, temps map[string][]byte, result string, eerr error) {
	marker := color.New(color.FgWhite, color.Bold).Sprint("=")
	literals := color.New(color.FgHiYellow).SprintFunc()
	errors := color.New(color.FgRed).SprintFunc()
	format := "%s %s\n"

	if eerr != nil {
		_, _ = fmt.Fprintf(w, format, marker, errors(eerr))
		return
	}

	value, ok := temps[result]
	if !ok {
		return // nothing to show: the line produced no value
	}

	// The tape itself, not a reading of it. A value is a run of bytes and nothing else —
	// there is no "true", no neutral value and no text to show, because true is a tape
	// holding 1, nothing is a tape of zeros and "a" is the tape holding 97. Writing the
	// decimal picked one of the three readings and hid the value behind it; printb, printd
	// and printc are how a program asks for a reading.
	_, _ = fmt.Fprintf(w, format, marker, literals(fmt.Sprintf("%v", value)))
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
func newLineReader(in io.Reader, out io.Writer) (lineReader, *History) {
	f, ok := in.(*os.File)
	if !ok || !isTTY(f) {
		return &scannerReader{scanner: bufio.NewScanner(in), out: out}, LoadHistory("")
	}

	// Without a home directory the history stays in memory; that is no reason to break the REPL.
	path, err := DefaultHistoryPath()
	if err != nil {
		path = ""
	}
	hist := LoadHistory(path)

	return newEditor(f, out, prompt, hist, func() (func(), error) {
		return enterRaw(f)
	}), hist
}

// Start runs a session, reading from in and writing everything it has to say to out — the
// prompt, the value of each line, and the errors that did not stop the session.
//
// Where it writes is the host's to decide, and it used to be os.Stdout from the inside,
// which is also why nothing could read a session back: the REPL is one of the two places the
// language proves itself and no test could see a single line of it.
func Start(in io.Reader, out io.Writer, loggers []string, tapeSize int) {
	tapeSize = byteutil.TapeSize(tapeSize)
	ev := evaluator.New(evaluator.NewEvaluatorOptions{
		Output:   out,
		TapeSize: tapeSize,
	})

	reader, hist := newLineReader(in, out)
	editing, interactive := reader.(*editor)

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		// os.Exit skips defers, so leave raw mode here before quitting.
		if interactive {
			editing.Restore()
		}
		_, _ = fmt.Fprintln(out, "Bye :)")
		os.Exit(0)
	}()

	var instsBuffer []emitter.Instruction
	// One session is one file typed a line at a time, and a parser is built per line, so the
	// struct directives are held here: a struct declared on one line has to still be known
	// on the next.
	directives := parser.NewDirectives()
	histWarned := false
	for {
		text, err := reader.ReadLine()
		if errors.Is(err, errInterrupt) { // Ctrl+C: drop the line, prompt again
			continue
		}
		if err != nil { // Ctrl+D or end of piped input
			if interactive {
				_, _ = fmt.Fprintln(out, "Bye :)")
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

		tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens(line.Bytes())
		if err != nil {
			_, _ = fmt.Fprintln(out, err)
			continue
		}
		if slices.Contains(loggers, "lexer") {
			if err := trace.Tokens(out, tokens); err != nil {
				_, _ = fmt.Fprintln(out, err)
			}
		}

		ast, err := parser.New(parser.NewParserOptions{
			Tokens:     tokens,
			TapeSize:   tapeSize,
			Directives: directives,
		}).Parse()
		if err != nil {
			_, _ = fmt.Fprintln(out, err)
			continue
		}
		// The phases return what they made; showing it is decided here.
		if slices.Contains(loggers, "parser") {
			if err := trace.AST(out, ast); err != nil {
				_, _ = fmt.Fprintln(out, err)
			}
		}

		program, err := emitter.New(emitter.NewEmitterOptions{
			TapeSize: tapeSize,
		}).EmitProgram(ast)
		if err != nil {
			_, _ = fmt.Fprintln(out, err)
			continue
		}
		if slices.Contains(loggers, "emitter") {
			if err := trace.Instructions(out, program.Instructions); err != nil {
				_, _ = fmt.Fprintln(out, err)
			}
		}

		// Append to buffer so defer from/to indices stay valid when calling later.
		offset := len(instsBuffer)
		instsBuffer = append(instsBuffer, program.Instructions...)

		// One expression at a time, so a line holding several of them prints each value
		// where it happens rather than all of them at the end.
		for _, expr := range program.Expressions {
			temps, err := ev.EvaluateRange(instsBuffer, uint64(offset+expr.From), uint64(offset+expr.To))
			render(out, temps, byteutil.ToHex(expr.Label), err)
			if err != nil {
				break
			}
		}
	}
}
