package builtin

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// tape builds a tape of the given width ending in the given bytes.
func tape(size int, values ...byte) []byte {
	return byteutil.PaddingTape(values, size)
}

// reel concatenates one tape per character, which is what a string is.
func reel(size int, text string) []byte {
	out := make([]byte, 0, len(text)*size)
	for i := 0; i < len(text); i++ {
		out = append(out, tape(size, text[i])...)
	}
	return out
}

func TestPrintFunction(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	value := []byte{0, 0, 42}

	PrintFunction(buf, value)

	if !bytes.Equal(buf.Bytes(), value) {
		t.Errorf("print wrote %v, want the bytes as they are: %v", buf.Bytes(), value)
	}
}

func TestEchoFunction(t *testing.T) {
	cases := []struct {
		name     string
		value    []byte
		tapeSize int
		want     string
	}{
		{name: "a reel", value: reel(8, "hello"), tapeSize: 8, want: "hello"},
		{name: "a reel of narrow tapes", value: reel(1, "hi"), tapeSize: 1, want: "hi"},
		{name: "one tape is one character", value: tape(8, 'a'), tapeSize: 8, want: "a"},
		{name: "a number is the character it stands for", value: tape(8, 65), tapeSize: 8, want: "A"},
		{name: "a one-byte tape", value: tape(1, 'z'), tapeSize: 1, want: "z"},
		{name: "space is printable", value: tape(8, ' '), tapeSize: 8, want: " "},
		{name: "empty input", value: nil, tapeSize: 8, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			EchoFunction(buf, tc.value, tc.tapeSize)
			if got := buf.String(); got != tc.want {
				t.Errorf("echo wrote %q, want %q", got, tc.want)
			}
		})
	}
}

// Everything echo produces has to reach the writer it was given. A tape of zeros used to
// go to the process's stdout instead, which in the playground means the browser console
// rather than the page.
func TestEchoWritesEverythingToTheWriter(t *testing.T) {
	for _, size := range []int{1, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			EchoFunction(buf, byteutil.FalseTape(size), size)
			if buf.Len() != 0 {
				t.Errorf("a tape of zeros wrote %q, want nothing", buf.String())
			}
		})
	}
}

// A value that is not printable ASCII comes out as its significant bytes rather than
// being dropped.
func TestEchoWithUnprintableValue(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	EchoFunction(buf, tape(8, 200), 8)

	if got := buf.Bytes(); len(got) != 1 || got[0] != 200 {
		t.Errorf("echo wrote %v, want the significant byte 200", got)
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

// A failing assertion decodes its message the same way echo does. Walking it with a
// stride of eight regardless of the tape size read past the end of a narrow reel and
// brought the whole program down with it.
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

func TestSignificantBytes(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{name: "drops leading zeros", in: []byte{0, 0, 1, 2}, want: []byte{1, 2}},
		{name: "keeps inner zeros", in: []byte{0, 1, 0, 2}, want: []byte{1, 0, 2}},
		{name: "all zeros has nothing", in: []byte{0, 0, 0}, want: nil},
		{name: "empty", in: nil, want: nil},
		{name: "nothing to drop", in: []byte{1, 2}, want: []byte{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := significantBytes(tc.in); !bytes.Equal(got, tc.want) {
				t.Errorf("significantBytes(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
