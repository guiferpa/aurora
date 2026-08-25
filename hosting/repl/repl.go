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
	"github.com/guiferpa/aurora/loader"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/module"
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
// outlives the line that filled it — a name, a shape declaration and a defer all have to
// still be there on the next one.
type Session struct {
	lexer   *lexer.Lexer
	parser  parser.Parser
	emitter emitter.Emitter

	ev       *evaluator.Evaluator
	reader   lineReader
	out      io.Writer
	tapeSize int

	// A parser is built per line, so the shape declarations are held here: a shape declared
	// on one line has to still be known on the next.
	declarations *parser.Declarations
	// Every line's instructions go into the same buffer, which is what keeps the range a
	// defer recorded valid when it is called on a later line.
	// blocks is the session so far. A session is a file typed slowly, so it only grows: a
	// scope written on one line is called from a later one, and what the name holds is the
	// number of its block.
	blocks []ir.Block

	// resolver finds the files a line imports. Nil is a session that takes no use line,
	// which is what a REPL with nowhere to read from is.
	resolver *resolver.Resolver
	// loaded is every module this session has run, by name. A module is a program: running
	// it twice would bind its names twice, and the second time is a conflict — so a use of
	// something already here is a use of what is already here, the way it is in any REPL
	// that imports.
	loaded map[module.ID]module.Module

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
		// A shape declared on one line has to still be known on the next, so what the
		// session remembers goes in with every line.
		Declarations: s.declarations,
	})
	if err != nil {
		return ir.Program{}, err
	}

	// What the line imports is loaded before the line runs, since the line is about to use
	// it. A line that imports nothing asks nothing of the world.
	if err := s.load(tree); err != nil {
		return ir.Program{}, err
	}

	program, err := s.emitter.EmitProgram(tree)
	if err != nil {
		return ir.Program{}, err
	}

	return program, nil
}

// imports reports whether a line names a module at all.
func imports(tree ast.AST) bool {
	for _, node := range tree.Nodes {
		if _, ok := node.(ast.UseDeclaration); ok {
			return true
		}
	}
	return false
}

// load brings in what a line imports, and checks the line against everything brought in.
//
// A module runs here the way it runs anywhere: its instructions go into the same buffer every
// line goes into, and it is evaluated as a range of it under its own name, so what it bound
// is in the environ that belongs to it. Being in the one buffer is what lets a scope from it
// be called on a later line — a defer records where its body sits, and the buffer is what
// those positions are into.
func (s *Session) load(tree ast.AST) error {
	if s.resolver == nil {
		if imports(tree) {
			return errors.New("this session has nowhere to read a module from")
		}
		return nil
	}

	// The entry has no path: a session is not a file. Nothing can import it either, which
	// is what that would have been for.
	modules, err := s.resolver.Dependencies("", tree)
	if err != nil {
		return err
	}

	fresh := make([]module.Module, 0, len(modules))
	for _, each := range modules {
		if _, done := s.loaded[each.ID]; !done {
			fresh = append(fresh, each)
		}
	}

	// The line is checked against everything this session knows, not only what arrived with
	// it: a name reached on line ten belongs to a module imported on line one.
	known := make([]module.Module, 0, len(s.loaded)+len(fresh)+1)
	for _, each := range s.loaded {
		known = append(known, each)
	}
	known = append(known, fresh...)
	if err := loader.Check(append(known, module.Module{ID: "", Tree: tree})); err != nil {
		return err
	}

	for _, each := range fresh {
		program, err := s.emitter.EmitProgram(each.Tree)
		if err != nil {
			return err
		}
		top := s.join(program)
		if _, err := s.ev.EvaluateBlocks(s.blocks, ir.Point{Block: top}, nil, string(each.ID)); err != nil {
			return err
		}
		s.loaded[each.ID] = each
		// What it returns is written down for the lines after this one: a session is a
		// file typed slowly, and the use line was already read when this module arrived.
		s.declarations.Import(string(each.ID), ast.Offer{Shapes: each.Tree.Shapes, Returns: each.Tree.Returns})
	}
	return nil
}

// evaluate runs the line's expressions one at a time, so a line holding several of them
// answers where each one happens rather than all of them at the end.
func (s *Session) evaluate(program ir.Program) {
	top := s.join(program)

	for at, expr := range program.Expressions {
		temps, err := s.ev.EvaluateBlocks(s.blocks, s.at(top, expr), s.stopsAt(top, program, at), "")
		render(s.out, temps, byteutil.ToHex(expr.Label), err)
		if err != nil {
			return
		}
	}
}

// join adds a line's blocks to the session and answers where that line begins.
//
// A session is a file typed slowly, and its blocks only ever grow: a scope written on one line
// is called from a later one, and what the name holds is the number of its block. Each line is
// compiled on its own and numbers from zero, so every line but the first moves.
func (s *Session) join(program ir.Program) ir.BlockID {
	top := ir.BlockID(len(s.blocks))
	s.blocks = append(s.blocks, ir.Shifted(program.Blocks, top)...)
	return top
}

// at answers where an expression of a line begins, among the session's blocks.
func (s *Session) at(top ir.BlockID, expr ir.Expression) ir.Point {
	return ir.Point{Block: expr.At.Block + top, At: expr.At.At}
}

// stopsAt answers where an expression of a line ends: where the next one begins. The last one
// on a line ends when the line does, and nothing stops it.
//
// Running one expression at a time is what puts each value next to the prints that came with
// it. Running the line whole and reading the values afterwards put every printed line first
// and the values after, in no order at all.
func (s *Session) stopsAt(top ir.BlockID, program ir.Program, at int) func(ir.Point) bool {
	if at+1 >= len(program.Expressions) {
		return nil
	}
	next := s.at(top, program.Expressions[at+1])
	return func(point ir.Point) bool { return point == next }
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
	// Resolver finds the files a use line names. Without one a session takes no imports.
	Resolver *resolver.Resolver
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
		resolver:     opts.Resolver,
		loaded:       make(map[module.ID]module.Module),
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
