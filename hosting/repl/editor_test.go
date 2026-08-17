package repl

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// Escape sequences as a terminal sends them.
const (
	keyUp     = "\x1b[A"
	keyDown   = "\x1b[B"
	keyRight  = "\x1b[C"
	keyLeft   = "\x1b[D"
	keyHome   = "\x1b[H"
	keyEnd    = "\x1b[F"
	keyDel    = "\x1b[3~"
	keyEnterS = "\r"
)

// readLine drives the editor with a canned keystroke stream. raw is nil: putting a real
// terminal in raw mode is the caller's job, so the editor is testable without a TTY.
func readLine(t *testing.T, hist *History, input string) (string, error) {
	t.Helper()
	e := newEditor(strings.NewReader(input), io.Discard, prompt, hist, nil)
	return e.ReadLine()
}

func TestEditorLineEditing(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"plain typing", "ident a = 1;" + keyEnterS, "ident a = 1;"},
		{"left arrows then insert", "ab" + keyLeft + keyLeft + "X" + keyEnterS, "Xab"},
		{"left then right", "ab" + keyLeft + keyLeft + keyRight + "X" + keyEnterS, "aXb"},
		{"right stops at end", "ab" + keyRight + keyRight + "X" + keyEnterS, "abX"},
		{"left stops at start", "ab" + keyLeft + keyLeft + keyLeft + "X" + keyEnterS, "Xab"},
		{"backspace at end", "abc\x7f" + keyEnterS, "ab"},
		{"backspace in the middle", "abc" + keyLeft + "\x7f" + keyEnterS, "ac"},
		{"backspace at start is a no-op", "ab" + keyHome + "\x7f" + keyEnterS, "ab"},
		{"delete key", "abc" + keyLeft + keyDel + keyEnterS, "ab"},
		{"home and end", "ab" + keyHome + "X" + keyEnd + "Y" + keyEnterS, "XabY"},
		{"ctrl+a and ctrl+e", "ab\x01X\x05Y" + keyEnterS, "XabY"},
		{"ctrl+d deletes forward when not empty", "abc" + keyLeft + "\x04" + keyEnterS, "ab"},
		{"utf-8 runes", "áé" + keyEnterS, "áé"},
		{"cursor moves by rune, not byte", "ãb" + keyLeft + keyLeft + "X" + keyEnterS, "Xãb"},
		{"line feed submits", "abc\n", "abc"},
		{"unknown escape is ignored", "a\x1b[Zb" + keyEnterS, "ab"},
		{"lone control chars are ignored", "a\x0bb" + keyEnterS, "ab"},
		{"ctrl+d at the end of the line is a no-op", "abc\x04" + keyEnterS, "abc"},
		{"delete at the end of the line is a no-op", "abc" + keyDel + keyEnterS, "abc"},
		// A terminal in application cursor mode sends ESC O A rather than ESC [ A, and the
		// numbered forms of home and end are what several terminals send instead of H and F.
		{"application mode arrows", "ab\x1bOD\x1bODX" + keyEnterS, "Xab"},
		{"numbered home and end", "ab\x1b[1~X\x1b[4~Y" + keyEnterS, "XabY"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readLine(t, LoadHistory(""), tt.input)
			if err != nil {
				t.Fatalf("ReadLine: %v", err)
			}
			if got != tt.want {
				t.Errorf("line = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditorCtrlCInterrupts(t *testing.T) {
	got, err := readLine(t, LoadHistory(""), "abc\x03")
	if !errors.Is(err, errInterrupt) {
		t.Fatalf("err = %v, want errInterrupt", err)
	}
	if got != "" {
		t.Errorf("line = %q, want empty", got)
	}
}

func TestEditorCtrlDOnEmptyLineIsEOF(t *testing.T) {
	if _, err := readLine(t, LoadHistory(""), "\x04"); !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestEditorEndOfInputIsEOF(t *testing.T) {
	if _, err := readLine(t, LoadHistory(""), "abc"); !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestEditorHistoryNavigation(t *testing.T) {
	newHist := func(t *testing.T) *History {
		t.Helper()
		h := LoadHistory("")
		for _, line := range []string{"ident a = 1;", "a + 1;"} {
			if err := h.Append(line); err != nil {
				t.Fatalf("append: %v", err)
			}
		}
		return h
	}

	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"up recalls the last entry", keyUp + keyEnterS, "a + 1;"},
		{"up twice reaches the oldest", keyUp + keyUp + keyEnterS, "ident a = 1;"},
		{"up stops at the oldest", keyUp + keyUp + keyUp + keyEnterS, "ident a = 1;"},
		{"down returns to the newest", keyUp + keyUp + keyDown + keyEnterS, "a + 1;"},
		{"down at the end keeps the current line", keyDown + "x;" + keyEnterS, "x;"},
		{"recalled line is editable", keyUp + "\x7f\x7f" + "2;" + keyEnterS, "a + 2;"},
		{"typed line is stashed and restored", "part" + keyUp + keyDown + keyEnterS, "part"},
		// Up at the oldest entry changes nothing but still takes the cursor to the end, which
		// is the one place a key that does nothing still moves something.
		{
			"up at the oldest leaves the cursor at the end",
			keyUp + keyUp + keyHome + keyUp + "X" + keyEnterS,
			"ident a = 1;X",
		},
		// Down with nothing ahead restores the stash, and the stash is only written on the
		// first Up — so this drops the line being typed. It is what the editor does today.
		{"down with nothing to recall drops the line", "part" + keyDown + keyEnterS, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readLine(t, newHist(t), tt.input)
			if err != nil {
				t.Fatalf("ReadLine: %v", err)
			}
			if got != tt.want {
				t.Errorf("line = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditorRedrawPlacesCursor(t *testing.T) {
	out := bytes.NewBuffer(nil)
	e := newEditor(strings.NewReader("abc"+keyLeft+keyLeft), out, prompt, LoadHistory(""), nil)
	_, _ = e.ReadLine()

	rendered := out.String()
	if !strings.Contains(rendered, "\r\x1b[K"+prompt+"abc") {
		t.Errorf("expected a full line redraw with the prompt, got %q", rendered)
	}
	// After two left arrows the cursor must be walked back two columns.
	if !strings.HasSuffix(rendered, "\x1b[2D") {
		t.Errorf("expected trailing cursor-back of 2 columns, got %q", rendered)
	}
}

func TestEditorRestoreIsSafeWithoutRawMode(t *testing.T) {
	e := newEditor(strings.NewReader(""), io.Discard, prompt, LoadHistory(""), nil)
	e.Restore()
	e.Restore()
}

func TestEditorEntersAndLeavesRawMode(t *testing.T) {
	entered, restored := 0, 0
	raw := func() (func(), error) {
		entered++
		return func() { restored++ }, nil
	}

	e := newEditor(strings.NewReader("a;"+keyEnterS), io.Discard, prompt, LoadHistory(""), raw)
	if _, err := e.ReadLine(); err != nil {
		t.Fatalf("ReadLine: %v", err)
	}

	if entered != 1 {
		t.Errorf("entered raw mode %d times, want 1", entered)
	}
	if restored != 1 {
		t.Errorf("restored terminal %d times, want 1", restored)
	}
}

func TestEditorRawModeFailurePropagates(t *testing.T) {
	want := errors.New("no tty")
	e := newEditor(strings.NewReader("a;"+keyEnterS), io.Discard, prompt, LoadHistory(""), func() (func(), error) {
		return nil, want
	})
	if _, err := e.ReadLine(); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}
