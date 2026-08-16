package byteutil

import "testing"

// A host that reads the size from outside the program gets it as text, and text can be
// anything. What it must never be is a reason to stop: a bad value falls back to the width
// the host starts from, which is not always the language's.
func TestParseTapeSize(t *testing.T) {
	const fallback = 32

	cases := []struct {
		name string
		text string
		want int
	}{
		{name: "a size", text: "16", want: 16},
		{name: "the narrowest", text: "1", want: 1},
		{name: "the widest", text: "32", want: 32},
		{name: "written with spaces around it", text: " 8 ", want: 8},

		{name: "nothing", text: "", want: fallback},
		{name: "not a number", text: "wide", want: fallback},
		{name: "a number no tape can be", text: "0", want: fallback},
		{name: "wider than the EVM word", text: "33", want: fallback},
		{name: "negative", text: "-1", want: fallback},
		// The fallback is the caller's, and it is not normalized on the way out: a host
		// that says 32 gets 32, where TapeSize would have answered with the language's 8.
		{name: "a number with something after it", text: "8px", want: fallback},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseTapeSize(tc.text, fallback); got != tc.want {
				t.Errorf("ParseTapeSize(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}
