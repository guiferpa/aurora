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

// session is one file typed a line at a time: what a line needs to be compiled and run, and
// what it leaves behind for the line after it.
//
// It is a type rather than a handful of variables inside a loop because everything here
// outlives the line that filled it — a name, a struct directive and a defer all have to
// still be there on the next one.
type session struct {
	ev       *evaluator.Evaluator
	out      io.Writer
	tapeSize int
	loggers  []string

	// A parser is built per line, so the struct directives are held here: a struct declared
	// on one line has to still be known on the next.
	directives *parser.Directives
	// Every line's instructions go into the same buffer, which is what keeps the range a
	// defer recorded valid when it is called on a later line.
	insts []emitter.Instruction

	hist       *History
	histWarned bool
}

// run compiles a line and runs it. An error is written and swallowed: a session survives a
// line that does not compile, which is most of what a REPL is for.
func (s *session) run(text string) {
	program, err := s.compile(text)
	if err != nil {
		_, _ = fmt.Fprintln(s.out, err)
		return
	}
	s.evaluate(program)
}

// compile takes the line through the three phases, showing what each one produced when -l
// asked for it.
func (s *session) compile(text string) (emitter.Program, error) {
	line := bytes.NewBufferString(text)

	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens(line.Bytes())
	if err != nil {
		return emitter.Program{}, err
	}
	s.trace("lexer", func(w io.Writer) error { return trace.Tokens(w, tokens) })

	ast, err := parser.New(parser.NewParserOptions{
		Tokens:     tokens,
		TapeSize:   s.tapeSize,
		Directives: s.directives,
	}).Parse()
	if err != nil {
		return emitter.Program{}, err
	}
	s.trace("parser", func(w io.Writer) error { return trace.AST(w, ast) })

	program, err := emitter.New(emitter.NewEmitterOptions{
		TapeSize: s.tapeSize,
	}).EmitProgram(ast)
	if err != nil {
		return emitter.Program{}, err
	}
	s.trace("emitter", func(w io.Writer) error { return trace.Instructions(w, program.Instructions) })

	return program, nil
}

// trace writes what a phase produced, when -l named that phase. The phases return what they
// made; showing it is decided here, in the host.
func (s *session) trace(phase string, write func(w io.Writer) error) {
	if !slices.Contains(s.loggers, phase) {
		return
	}
	if err := write(s.out); err != nil {
		_, _ = fmt.Fprintln(s.out, err)
	}
}

// evaluate runs the line's expressions one at a time, so a line holding several of them
// answers where each one happens rather than all of them at the end.
func (s *session) evaluate(program emitter.Program) {
	offset := len(s.insts)
	s.insts = append(s.insts, program.Instructions...)

	for _, expr := range program.Expressions {
		temps, err := s.ev.EvaluateRange(s.insts, uint64(offset+expr.From), uint64(offset+expr.To))
		render(s.out, temps, byteutil.ToHex(expr.Label), err)
		if err != nil {
			return
		}
	}
}

// remember records the line before it is evaluated, so a line that fails to compile is still
// recallable. A history that cannot be written is said once, on stderr — it is about the
// environment and not about the session, which carries on either way.
func (s *session) remember(text string) {
	if err := s.hist.Append(text); err != nil && !s.histWarned {
		s.histWarned = true
		_, _ = fmt.Fprintf(os.Stderr, "warning: could not write history: %v\n", err)
	}
}

// Start runs a session, reading from in and writing everything it has to say to out — the
// prompt, the value of each line, and the errors that did not stop the session.
//
// Where it writes is the host's to decide, and it used to be os.Stdout from the inside,
// which is also why nothing could read a session back: the REPL is one of the two places the
// language proves itself and no test could see a single line of it.
func Start(in io.Reader, out io.Writer, loggers []string, tapeSize int) {
	tapeSize = byteutil.TapeSize(tapeSize)
	reader, hist := newLineReader(in, out)

	s := &session{
		ev: evaluator.New(evaluator.NewEvaluatorOptions{
			Output:   out,
			TapeSize: tapeSize,
		}),
		out:        out,
		tapeSize:   tapeSize,
		loggers:    loggers,
		directives: parser.NewDirectives(),
		hist:       hist,
	}

	editing, interactive := reader.(*editor)
	leaveOnInterrupt(out, func() {
		// os.Exit skips defers, so leave raw mode here before quitting.
		if interactive {
			editing.Restore()
		}
	})

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

		s.remember(text)
		s.run(text)
	}
}

// leaveOnInterrupt says goodbye on Ctrl+C, after giving the terminal back.
func leaveOnInterrupt(out io.Writer, restore func()) {
	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		<-csig
		restore()
		_, _ = fmt.Fprintln(out, "Bye :)")
		os.Exit(0)
	}()
}
