package repl

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The REPL writes the tape, not a reading of it. A value is a run of bytes, and decimal was
// one of the three readings a program can ask for — showing it hid the value behind a
// choice the line never made.
func TestRenderWritesTheTape(t *testing.T) {
	cases := []struct {
		name  string
		value []byte
		want  string
	}{
		{name: "a number", value: []byte{0, 0, 0, 0, 0, 0, 0, 42}, want: "[0 0 0 0 0 0 0 42]"},
		// true is a tape holding 1, so there is no "true" to write.
		{name: "true", value: []byte{0, 0, 0, 0, 0, 0, 0, 1}, want: "[0 0 0 0 0 0 0 1]"},
		{name: "the neutral value", value: []byte{0, 0, 0, 0, 0, 0, 0, 0}, want: "[0 0 0 0 0 0 0 0]"},
		// "hi" is the tape holding its bytes, and nothing says it was written as text.
		{name: "text", value: []byte{0, 0, 0, 0, 0, 0, 104, 105}, want: "[0 0 0 0 0 0 104 105]"},
		// A run of tapes is written whole, so a shape shows its fields end to end.
		{
			name:  "a run of tapes",
			value: []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2},
			want:  "[0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 2]",
		},
		// A value that is not a whole number of tapes used to fail to render at all.
		{name: "not a whole tape", value: []byte{1, 2, 3}, want: "[1 2 3]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := bytes.NewBuffer(nil)
			render(out, map[string][]byte{"01": tc.value}, "01", nil)

			if got := out.String(); !strings.Contains(got, tc.want) {
				t.Errorf("rendered %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestRenderWritesAnError(t *testing.T) {
	out := bytes.NewBuffer(nil)
	render(out, nil, "01", errors.New("identifier x not found"))

	if got := out.String(); !strings.Contains(got, "identifier x not found") {
		t.Errorf("rendered %q, want the error", got)
	}
}

// A line that produced no value shows nothing rather than an empty answer.
func TestRenderSaysNothingWithoutAValue(t *testing.T) {
	out := bytes.NewBuffer(nil)
	render(out, map[string][]byte{}, "01", nil)

	if got := out.String(); got != "" {
		t.Errorf("rendered %q, want nothing", got)
	}
}
