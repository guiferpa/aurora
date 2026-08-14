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

// reel concatenates one tape per character, which is what a string is.
func reel(size int, text string) []byte {
	out := make([]byte, 0, len(text)*size)
	for _, char := range text {
		out = append(out, tape(size, uint64(char))...)
	}
	return out
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
			// 514 is beyond a byte, which is the point: a character is the number the
			// tape holds, not the byte at its end.
			name:  "514 is a letter outside ASCII",
			value: tape(8, 514),
			bytes: "[0 0 0 0 0 0 2 2]",
			decs:  "514",
			chars: "Ȃ",
		},
		{
			name:  "a reel reads tape by tape",
			value: reel(8, "hi"),
			bytes: "[0 0 0 0 0 0 0 104 0 0 0 0 0 0 0 105]",
			decs:  "104 105",
			chars: "hi",
		},
		{
			name:  "an accented reel survives whole",
			value: reel(8, "café"),
			decs:  "99 97 102 233",
			chars: "café",
		},
		{
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

// A character outside ASCII used to be dropped: the reader kept only the last byte of the
// tape and wrote it when it fell between 32 and 126.
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
			if got := TextOf(reel(8, tc.text), 8); got != tc.text {
				t.Errorf("got %q, want %q", got, tc.text)
			}
		})
	}
}

func TestTextOfAcrossTapeSizes(t *testing.T) {
	// A one-byte tape only names characters up to 255, so this is ASCII.
	if got := TextOf(reel(1, "hi"), 1); got != "hi" {
		t.Errorf("one-byte tapes: got %q, want %q", got, "hi")
	}
	// Two bytes reach beyond it.
	if got := TextOf(reel(2, "café"), 2); got != "café" {
		t.Errorf("two-byte tapes: got %q, want %q", got, "café")
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

// A number naming no character has nothing to write, and must not corrupt what surrounds
// it in the output.
func TestTextOfSkipsWhatNamesNoCharacter(t *testing.T) {
	value := make([]byte, 0)
	value = append(value, tape(8, 'a')...)
	value = append(value, tape(8, 0xD800)...) // a surrogate half: not a character on its own
	value = append(value, tape(8, 'b')...)

	if got := TextOf(value, 8); got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestAssertFunction(t *testing.T) {
	passed, err := AssertFunction(byteutil.TrueTape(8), reel(8, "unused"), 8)
	if !passed || err != nil {
		t.Errorf("a true condition passes: got (%v, %v)", passed, err)
	}

	passed, err = AssertFunction(byteutil.FalseTape(8), reel(8, "boom"), 8)
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
			_, err := AssertFunction(byteutil.FalseTape(size), reel(size, "boom"), size)
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
	_, err := AssertFunction(byteutil.FalseTape(8), nil, 8)
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
		{name: "a reel of three", value: reel(8, "abc"), tapeSize: 8, want: 3},
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
