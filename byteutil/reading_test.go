package byteutil

import (
	"strings"
	"testing"
)

// tape builds a tape of the given width holding a number.
func tape(size int, value uint64) []byte {
	return PaddingTape(FromUint64(value), size)
}

// text builds the tape a text literal compiles to: its bytes, right aligned.
func text(size int, s string) []byte {
	return PaddingTape([]byte(s), size)
}

// A tape is bytes, and reading it as text or as a number is reading those same bytes. Nothing
// in a value says which reading was meant, which is the whole point of having three of them.
func TestTheReadingsOfATape(t *testing.T) {
	cases := []struct {
		name    string
		value   []byte
		decimal string
		chars   string
	}{
		{name: "44 is a comma", value: tape(8, 44), decimal: "44", chars: ","},
		{
			// 18537 is the bytes 72 and 105, which spell Hi.
			name:    "a number is a word is a tape",
			value:   tape(8, 18537),
			decimal: "18537",
			chars:   "Hi",
		},
		{name: "a word is one tape", value: text(8, "hi"), decimal: "26729", chars: "hi"},
		{
			// Which is what keeps an accent whole: café is five UTF-8 bytes.
			name:    "an accented word survives whole",
			value:   text(8, "café"),
			decimal: "426835887017",
			chars:   "café",
		},
		{name: "a tape of zeros is the neutral value", value: FalseTape(8), decimal: "0", chars: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DecimalOf(tc.value, 8); got != tc.decimal {
				t.Errorf("as a number it reads %q, want %q", got, tc.decimal)
			}
			if got := TextOf(tc.value, 8); got != tc.chars {
				t.Errorf("as text it reads %q, want %q", got, tc.chars)
			}
		})
	}
}

// Text outside ASCII is its UTF-8 bytes, and reading gives them back unchanged.
func TestTextOfKeepsCharactersBeyondASCII(t *testing.T) {
	for _, word := range []string{"café", "ação", "λόγος", "привет", "日本語"} {
		t.Run(word, func(t *testing.T) {
			if got := TextOf(text(32, word), 32); got != word {
				t.Errorf("read %q, want %q", got, word)
			}
		})
	}
}

// The width changes how much fits, never what the bytes mean.
func TestTextOfAcrossTapeSizes(t *testing.T) {
	for _, size := range []int{1, 2, 8, 16, 32} {
		t.Run(strings.Repeat("x", size), func(t *testing.T) {
			if got := TextOf(text(size, "a"), size); got != "a" {
				t.Errorf("at %d bytes it reads %q, want a", size, got)
			}
		})
	}
}

// A run of tapes is read tape by tape, which is what lets a word held across several of them
// still read as one word.
func TestReadingARunOfTapes(t *testing.T) {
	run := append(text(8, "hi"), text(8, "!")...)

	if got := TextOf(run, 8); got != "hi!" {
		t.Errorf("as text the run reads %q, want hi!", got)
	}
	if got := DecimalOf(run, 8); got != "26729 33" {
		t.Errorf("as numbers the run reads %q, want one per tape", got)
	}
}

// Nothing to read is nothing, not a zero: a caller showing this prints an empty line rather
// than a number nobody asked for.
func TestReadingAnEmptyValue(t *testing.T) {
	if got := TextOf(nil, 8); got != "" {
		t.Errorf("as text it reads %q", got)
	}
	if got := DecimalOf(nil, 8); got != "" {
		t.Errorf("as a number it reads %q", got)
	}
}

// Bytes that are not text have nothing to say as text, and the value is still whatever it is.
func TestTextOfBytesThatAreNotText(t *testing.T) {
	if got := TextOf([]byte{0xff, 0xfe}, 8); got != "" {
		t.Errorf("read %q from bytes that are not UTF-8", got)
	}
}
