package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"
)

// errInterrupt is returned by ReadLine when the user presses Ctrl+C: the line in progress
// is discarded and the REPL prompts again, as a shell does.
var errInterrupt = errors.New("interrupted")

// Control characters handled by the editor.
const (
	keyCtrlA     = 0x01
	keyCtrlC     = 0x03
	keyCtrlD     = 0x04
	keyCtrlE     = 0x05
	keyCtrlH     = 0x08
	keyLineFeed  = 0x0a
	keyEnter     = 0x0d
	keyEscape    = 0x1b
	keyBackspace = 0x7f
)

// editor reads a line keystroke by keystroke, keeping a rune buffer and a cursor so the
// arrow keys can move inside the text and browse the history.
//
// It works over plain io.Reader/io.Writer and is tested that way. Putting the terminal in
// raw mode is the job of the raw hook (nil in tests), which is held only while reading.
type editor struct {
	in     *bufio.Reader
	out    io.Writer
	prompt string
	hist   *History
	raw    func() (func(), error)

	mu      sync.Mutex
	restore func()
}

func newEditor(in io.Reader, out io.Writer, prompt string, hist *History, raw func() (func(), error)) *editor {
	return &editor{
		in:     bufio.NewReader(in),
		out:    out,
		prompt: prompt,
		hist:   hist,
		raw:    raw,
	}
}

// Restore leaves raw mode if the editor is currently in it. It is safe to call from a
// signal handler, which is the one path that can end the process while a line is being read.
func (e *editor) Restore() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.restore != nil {
		e.restore()
		e.restore = nil
	}
}

func (e *editor) setRestore(f func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.restore = f
}

// line is what is being typed and everything that moves inside it: the runes, where the
// cursor sits, how far back the history has been walked, and the line that was stashed when
// that walk began.
//
// It exists so that a key is a small function with a name — insert, backspace, historyBack —
// instead of three lines of slice arithmetic in the middle of a switch, which is what took
// ReadLine to 35 of cognitive complexity.
type line struct {
	buf  []rune
	pos  int
	hist *History
	// histIdx walks the history; hist.Len() means "the line being typed".
	histIdx int
	stash   string
}

// text is the line as it would be submitted.
func (l *line) text() string { return string(l.buf) }

// insert puts a rune where the cursor is and steps over it.
func (l *line) insert(r rune) {
	l.buf = append(l.buf, 0)
	copy(l.buf[l.pos+1:], l.buf[l.pos:])
	l.buf[l.pos] = r
	l.pos++
}

// backspace removes the rune before the cursor.
func (l *line) backspace() {
	if l.pos == 0 {
		return
	}
	l.buf = append(l.buf[:l.pos-1], l.buf[l.pos:]...)
	l.pos--
}

// deleteUnder removes the rune the cursor is on, and does nothing at the end of the line.
func (l *line) deleteUnder() {
	if l.pos < len(l.buf) {
		l.buf = append(l.buf[:l.pos], l.buf[l.pos+1:]...)
	}
}

func (l *line) left() {
	if l.pos > 0 {
		l.pos--
	}
}

func (l *line) right() {
	if l.pos < len(l.buf) {
		l.pos++
	}
}

func (l *line) toStart() { l.pos = 0 }

func (l *line) toEnd() { l.pos = len(l.buf) }

// historyBack moves one entry towards the past, stashing the line being typed on the first
// move. At the oldest entry it changes nothing and still takes the cursor to the end.
func (l *line) historyBack() {
	if l.histIdx <= 0 {
		l.toEnd()
		return
	}
	if l.histIdx == l.hist.Len() {
		l.stash = l.text()
	}
	l.histIdx--
	l.buf = []rune(l.hist.At(l.histIdx))
	l.toEnd()
}

// historyForward moves one entry towards the present, restoring the stashed line at the end
// of the walk.
func (l *line) historyForward() {
	if l.histIdx >= l.hist.Len() {
		l.buf = []rune(l.stash)
		l.toEnd()
		return
	}
	l.histIdx++
	l.buf = []rune(l.stash)
	if l.histIdx < l.hist.Len() {
		l.buf = []rune(l.hist.At(l.histIdx))
	}
	l.toEnd()
}

// The keys that change the line without ending it. Ctrl+D is here for the line it deletes
// from; on an empty line it ends the session instead, which ReadLine answers before looking
// here.
var editKeys = map[rune]func(l *line){
	keyBackspace: (*line).backspace,
	keyCtrlH:     (*line).backspace,
	keyCtrlD:     (*line).deleteUnder,
	keyCtrlA:     (*line).toStart,
	keyCtrlE:     (*line).toEnd,
}

// The same, for the keys that arrive as an escape sequence.
var escapeKeys = map[escKey]func(l *line){
	escUp:     (*line).historyBack,
	escDown:   (*line).historyForward,
	escLeft:   (*line).left,
	escRight:  (*line).right,
	escHome:   (*line).toStart,
	escEnd:    (*line).toEnd,
	escDelete: (*line).deleteUnder,
}

// ReadLine reads one line. It returns io.EOF on Ctrl+D with an empty buffer and
// errInterrupt on Ctrl+C.
func (e *editor) ReadLine() (string, error) {
	if e.raw != nil {
		restore, err := e.raw()
		if err != nil {
			return "", err
		}
		e.setRestore(restore)
		defer e.Restore()
	}

	l := &line{buf: make([]rune, 0), hist: e.hist, histIdx: e.hist.Len()}
	e.redraw(l)

	for {
		r, _, err := e.in.ReadRune()
		if err != nil {
			return "", err
		}

		switch {
		case r == keyEnter || r == keyLineFeed:
			return e.end(l.text(), nil)
		case r == keyCtrlC:
			return e.end("", errInterrupt)
		case r == keyCtrlD && len(l.buf) == 0:
			return e.end("", io.EOF)
		case r == keyEscape:
			// A lone ESC blocks until the next key arrives; acceptable for a REPL.
			apply(escapeKeys[e.readEscape()], l)
		default:
			e.press(r, l)
		}

		e.redraw(l)
	}
}

// press applies a key that does not end the line: one of the editing keys, the rune itself,
// or nothing at all.
func (e *editor) press(r rune, l *line) {
	if edit, ok := editKeys[r]; ok {
		edit(l)
		return
	}
	if r < 0x20 {
		return // other control characters are ignored
	}
	l.insert(r)
}

// apply runs a key that may not be there: an escape sequence the editor does not know reads
// as no key, and no key does nothing.
func apply(edit func(l *line), l *line) {
	if edit != nil {
		edit(l)
	}
}

// end closes the line off on the terminal and answers with what it was.
func (e *editor) end(text string, err error) (string, error) {
	_, _ = fmt.Fprint(e.out, "\r\n")
	return text, err
}

type escKey int

const (
	escNone escKey = iota
	escUp
	escDown
	escRight
	escLeft
	escHome
	escEnd
	escDelete
)

// readEscape consumes the rest of an escape sequence already started by ESC.
// Handles CSI ("ESC [ ... final") and the application-mode form ("ESC O final").
func (e *editor) readEscape() escKey {
	r, _, err := e.in.ReadRune()
	if err != nil {
		return escNone
	}

	switch r {
	case 'O': // application cursor mode: ESC O A/B/C/D/H/F
		final, _, err := e.in.ReadRune()
		if err != nil {
			return escNone
		}
		return finalToKey(final, "")
	case '[':
	default:
		return escNone
	}

	// CSI: parameter bytes until a final byte in the range @ to ~.
	params := make([]rune, 0, 4)
	for {
		c, _, err := e.in.ReadRune()
		if err != nil {
			return escNone
		}
		if c >= '@' && c <= '~' {
			return finalToKey(c, string(params))
		}
		params = append(params, c)
	}
}

func finalToKey(final rune, params string) escKey {
	switch final {
	case 'A':
		return escUp
	case 'B':
		return escDown
	case 'C':
		return escRight
	case 'D':
		return escLeft
	case 'H':
		return escHome
	case 'F':
		return escEnd
	case '~':
		switch params {
		case "1", "7":
			return escHome
		case "3":
			return escDelete
		case "4", "8":
			return escEnd
		}
	}
	return escNone
}

// redraw rewrites the whole line: carriage return, erase to end of line, prompt, buffer,
// then walks the cursor back when it is not at the end.
//
// Lines longer than the terminal width are not reflowed (known limitation).
func (e *editor) redraw(l *line) {
	_, _ = fmt.Fprintf(e.out, "\r\x1b[K%s%s", e.prompt, l.text())
	if back := len(l.buf) - l.pos; back > 0 {
		_, _ = fmt.Fprintf(e.out, "\x1b[%dD", back)
	}
}
