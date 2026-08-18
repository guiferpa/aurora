package repl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
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
// outlives the line that filled it — a name, a struct declaration and a defer all have to
// still be there on the next one.
type Session struct {
	lexer   *lexer.Lexer
	parser  parser.Parser
	emitter emitter.Emitter

	ev       *evaluator.Evaluator
	reader   lineReader
	out      io.Writer
	tapeSize int

	// A parser is built per line, so the struct declarations are held here: a struct declared
	// on one line has to still be known on the next.
	declarations *parser.Declarations
	// Every line's instructions go into the same buffer, which is what keeps the range a
	// defer recorded valid when it is called on a later line.
	insts []ir.Instruction

	hist       *History
	histWarned bool
}

// run compiles a line and runs it. An error is written and swallowed: a session survives a
// line that does not compile, which is most of what a REPL is for.
func (s *Session) run(text string) {
	program, err := s.compile(text)
	if err != nil {
		_, _ = fmt.Fprintln(s.out, err)
		return
	}
	s.evaluate(program)
}

// compile takes the line through the three phases, showing what each one produced when -l
// asked for it.
func (s *Session) compile(text string) (ir.Program, error) {
	line := bytes.NewBufferString(text)

	tokens, err := s.lexer.GetFilledTokens(line.Bytes())
	if err != nil {
		return ir.Program{}, err
	}

	tree, err := s.parser.Parse(parser.ParseInput{
		Tokens:   tokens,
		TapeSize: s.tapeSize,
		// A struct declared on one line has to still be known on the next, so what the
		// session remembers goes in with every line.
		Declarations: s.declarations,
	})
	if err != nil {
		return ir.Program{}, err
	}

	program, err := s.emitter.EmitProgram(tree)
	if err != nil {
		return ir.Program{}, err
	}

	return program, nil
}

// evaluate runs the line's expressions one at a time, so a line holding several of them
// answers where each one happens rather than all of them at the end.
func (s *Session) evaluate(program ir.Program) {
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
func (s *Session) remember(text string) {
	if err := s.hist.Append(text); err != nil && !s.histWarned {
		s.histWarned = true
		_, _ = fmt.Fprintf(os.Stderr, "warning: could not write history: %v\n", err)
	}
}

// NewSessionOptions is what a session is made of. The phases and the evaluator arrive built,
// from cmd/aurora, which is where the flags that decide how to build them are read.
type NewSessionOptions struct {
	Lexer     *lexer.Lexer
	Parser    parser.Parser
	Emitter   emitter.Emitter
	Evaluator *evaluator.Evaluator
	// In is where lines are typed; Out receives everything the session has to say — the
	// prompt, the value of each line, and the errors that did not stop it.
	In  io.Reader
	Out io.Writer
	// TapeSize is the width in bytes of every value, the same one the phases were built with.
	TapeSize int
}

// NewSession builds a session from what it was handed.
//
// One evaluator lasts the whole session, unlike a command that runs one program: what a line
// binds has to still be there on the next one, and so does the scope a line deferred.
func NewSession(opts NewSessionOptions) *Session {
	tapeSize := byteutil.TapeSize(opts.TapeSize)
	reader, hist := newLineReader(opts.In, opts.Out)

	return &Session{
		lexer:        opts.Lexer,
		parser:       opts.Parser,
		emitter:      opts.Emitter,
		ev:           opts.Evaluator,
		reader:       reader,
		out:          opts.Out,
		tapeSize:     tapeSize,
		declarations: parser.NewDeclarations(),
		hist:         hist,
	}
}

// Start reads line after line until there are no more, and is where a session spends its life.
//
// Where it writes is the host's to decide, and it used to be os.Stdout from the inside, which
// is also why nothing could read a session back: the REPL is one of the two places the
// language proves itself and no test could see a single line of it.
func (s *Session) Start() {
	out, reader := s.out, s.reader

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
