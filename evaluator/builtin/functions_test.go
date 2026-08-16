package builtin

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// tape builds a tape of the given width holding a number.
func tape(size int, value uint64) []byte {
	return byteutil.PaddingTape(byteutil.FromUint64(value), size)
}

// text builds the tape a text literal compiles to: its bytes, right aligned.
func text(size int, s string) []byte {
	return byteutil.PaddingTape([]byte(s), size)
}

// The three builtins are three readings of the same tape. These are the readings named in
// their own documentation, so they are the ones worth pinning.
func TestTheThreeReadingsOfATape(t *testing.T) {
	cases := []struct {
		name  string
		value []byte
		bytes string
		decs  string
		chars string
	}{
		{
			name:  "44 is a comma",
			value: tape(8, 44),
			bytes: "[0 0 0 0 0 0 0 44]",
			decs:  "44",
			chars: ",",
		},
		{
			// A number and a word are the same tape: 18537 is the bytes 72 and 105, which
			// spell Hi. Nothing in the value says which reading was meant.
			name:  "a number is a word is a tape",
			value: tape(8, 18537),
			bytes: "[0 0 0 0 0 0 72 105]",
			decs:  "18537",
			chars: "Hi",
		},
		{
			name:  "a word is one tape",
			value: text(8, "hi"),
			bytes: "[0 0 0 0 0 0 104 105]",
			decs:  "26729",
			chars: "hi",
		},
		{
			// Which is what keeps an accent whole: café is five UTF-8 bytes.
			name:  "an accented word survives whole",
			value: text(8, "café"),
			bytes: "[0 0 0 99 97 102 195 169]",
			decs:  "426835887017",
			chars: "café",
		},
		{
			// A number is bytes too, and the byte 44 is a comma. Reading it back as text is
			// reading those bytes.
			name:  "a tape of zeros is the neutral value",
			value: byteutil.FalseTape(8),
			bytes: "[0 0 0 0 0 0 0 0]",
			decs:  "0",
			chars: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.bytes != "" {
				buf := bytes.NewBuffer(nil)
				PrintBytesFunction(buf, tc.value)
				if got := strings.TrimSuffix(buf.String(), "\n"); got != tc.bytes {
					t.Errorf("printb wrote %q, want %q", got, tc.bytes)
				}
			}

			buf := bytes.NewBuffer(nil)
			PrintDecimalFunction(buf, tc.value, 8)
			if got := strings.TrimSuffix(buf.String(), "\n"); got != tc.decs {
				t.Errorf("printd wrote %q, want %q", got, tc.decs)
			}

			buf = bytes.NewBuffer(nil)
			PrintCharsFunction(buf, tc.value, 8)
			if got := strings.TrimSuffix(buf.String(), "\n"); got != tc.chars {
				t.Errorf("printc wrote %q, want %q", got, tc.chars)
			}
		})
	}
}

// Text outside ASCII is its UTF-8 bytes, and reading gives them back unchanged.
func TestTextOfKeepsCharactersBeyondASCII(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{name: "accents", text: "café"},
		{name: "cedilla", text: "ação"},
		{name: "greek", text: "λόγος"},
		{name: "cyrillic", text: "привет"},
		{name: "cjk", text: "日本語"},
		{name: "emoji", text: "🌅"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TextOf(text(32, tc.text), 32); got != tc.text {
				t.Errorf("got %q, want %q", got, tc.text)
			}
		})
	}
}

func TestTextOfAcrossTapeSizes(t *testing.T) {
	// A tape holds as many bytes as it is wide, and text is bytes.
	if got := TextOf(text(2, "hi"), 2); got != "hi" {
		t.Errorf("two-byte tapes: got %q, want %q", got, "hi")
	}
	if got := TextOf(text(8, "café"), 8); got != "café" {
		t.Errorf("eight-byte tapes: got %q, want %q", got, "café")
	}
}

// A run of tapes reads as one word: the zeros that pad each tape are dropped, so a struct
// of one-character fields still spells something.
func TestTextOfAcrossARunOfTapes(t *testing.T) {
	run := make([]byte, 0, 16)
	run = append(run, text(8, "a")...)
	run = append(run, text(8, "b")...)

	if got := TextOf(run, 8); got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestDecimalOfAcrossTapeSizes(t *testing.T) {
	for _, size := range []int{1, 2, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			if got := DecimalOf(tape(size, 7), size); got != "7" {
				t.Errorf("got %q, want %q", got, "7")
			}
		})
	}
}

func TestReadingAnEmptyValue(t *testing.T) {
	if got := TextOf(nil, 8); got != "" {
		t.Errorf("TextOf(nil) = %q, want empty", got)
	}
	if got := DecimalOf(nil, 8); got != "" {
		t.Errorf("DecimalOf(nil) = %q, want empty", got)
	}
}

// Bytes that are not text have nothing to write. The value is still whatever it is — this
// is a reading, and one reading answering nothing says nothing about the others.
func TestTextOfBytesThatAreNotText(t *testing.T) {
	if got := TextOf([]byte{0xff, 0xfe}, 8); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

func TestAssertFunction(t *testing.T) {
	passed, err := AssertFunction(byteutil.TrueTape(8), "unused")
	if !passed || err != nil {
		t.Errorf("a true condition passes: got (%v, %v)", passed, err)
	}

	passed, err = AssertFunction(byteutil.FalseTape(8), "boom")
	if passed {
		t.Error("a false condition fails")
	}
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to carry the message", err)
	}
}

// A failing assertion reads its message the same way printc does. Walking it with a stride
// of eight regardless of the tape size read past the end of a narrow reel and brought the
// whole program down with it.
func TestAssertFunctionWithNarrowTapes(t *testing.T) {
	for _, size := range []int{1, 2, 4, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			_, err := AssertFunction(byteutil.FalseTape(size), "boom")
			if err == nil {
				t.Fatal("expected a failure")
			}
			if !strings.Contains(err.Error(), "boom") {
				t.Errorf("error = %q, want it to carry the message", err)
			}
		})
	}
}

func TestAssertFunctionWithNoMessage(t *testing.T) {
	_, err := AssertFunction(byteutil.FalseTape(8), "")
	if err == nil {
		t.Fatal("expected a failure")
	}
	if !strings.Contains(err.Error(), "assertion failed") {
		t.Errorf("error = %q, want it to still say what happened", err)
	}
}

func TestTapesOf(t *testing.T) {
	cases := []struct {
		name     string
		value    []byte
		tapeSize int
		want     int
	}{
		{name: "one tape", value: tape(8, 1), tapeSize: 8, want: 1},
		{name: "a run of three", value: append(append(tape(8, 1), tape(8, 2)...), tape(8, 3)...), tapeSize: 8, want: 3},
		{name: "narrower than a tape is still one", value: []byte{1, 2}, tapeSize: 8, want: 1},
		{name: "not a whole number of tapes is one", value: make([]byte, 9), tapeSize: 8, want: 1},
		{name: "empty", value: nil, tapeSize: 8, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(tapesOf(tc.value, tc.tapeSize)); got != tc.want {
				t.Errorf("got %d tapes, want %d", got, tc.want)
			}
		})
	}
}
