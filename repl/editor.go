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

	buf := make([]rune, 0)
	pos := 0
	// histIdx walks the history; hist.Len() means "the line being typed".
	histIdx := e.hist.Len()
	stash := ""

	e.redraw(buf, pos)

	for {
		r, _, err := e.in.ReadRune()
		if err != nil {
			return "", err
		}

		switch r {
		case keyEnter, keyLineFeed:
			_, _ = fmt.Fprint(e.out, "\r\n")
			return string(buf), nil

		case keyCtrlC:
			_, _ = fmt.Fprint(e.out, "\r\n")
			return "", errInterrupt

		case keyCtrlD:
			if len(buf) == 0 {
				_, _ = fmt.Fprint(e.out, "\r\n")
				return "", io.EOF
			}
			if pos < len(buf) {
				buf = append(buf[:pos], buf[pos+1:]...)
			}

		case keyBackspace, keyCtrlH:
			if pos > 0 {
				buf = append(buf[:pos-1], buf[pos:]...)
				pos--
			}

		case keyCtrlA:
			pos = 0

		case keyCtrlE:
			pos = len(buf)

		case keyEscape:
			// A lone ESC blocks until the next key arrives; acceptable for a REPL.
			switch e.readEscape() {
			case escUp:
				buf, pos, histIdx, stash = e.historyBack(buf, histIdx, stash)
			case escDown:
				buf, pos, histIdx = e.historyForward(histIdx, stash)
			case escLeft:
				if pos > 0 {
					pos--
				}
			case escRight:
				if pos < len(buf) {
					pos++
				}
			case escHome:
				pos = 0
			case escEnd:
				pos = len(buf)
			case escDelete:
				if pos < len(buf) {
					buf = append(buf[:pos], buf[pos+1:]...)
				}
			}

		default:
			if r < 0x20 {
				continue // other control characters are ignored
			}
			buf = append(buf, 0)
			copy(buf[pos+1:], buf[pos:])
			buf[pos] = r
			pos++
		}

		e.redraw(buf, pos)
	}
}

// historyBack moves one entry towards the past, stashing the line being typed on the first move.
func (e *editor) historyBack(buf []rune, histIdx int, stash string) ([]rune, int, int, string) {
	if histIdx <= 0 {
		return buf, len(buf), histIdx, stash
	}
	if histIdx == e.hist.Len() {
		stash = string(buf)
	}
	histIdx--
	next := []rune(e.hist.At(histIdx))
	return next, len(next), histIdx, stash
}

// historyForward moves one entry towards the present, restoring the stashed line at the end.
func (e *editor) historyForward(histIdx int, stash string) ([]rune, int, int) {
	if histIdx >= e.hist.Len() {
		return []rune(stash), len([]rune(stash)), histIdx
	}
	histIdx++
	next := []rune(stash)
	if histIdx < e.hist.Len() {
		next = []rune(e.hist.At(histIdx))
	}
	return next, len(next), histIdx
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
func (e *editor) redraw(buf []rune, pos int) {
	_, _ = fmt.Fprintf(e.out, "\r\x1b[K%s%s", e.prompt, string(buf))
	if back := len(buf) - pos; back > 0 {
		_, _ = fmt.Fprintf(e.out, "\x1b[%dD", back)
	}
}
