package printer

import (
	"errors"
	"strings"
	"testing"
)

// tape builds a tape of eight bytes holding a number.
func tape(value byte) []byte {
	return []byte{0, 0, 0, 0, 0, 0, 0, value}
}

// The three printers are three readings of the same tape, which is the whole difference
// between printb, printc and printd.
func TestTheThreePrintersReadTheSameTape(t *testing.T) {
	value := tape(44)

	cases := []struct {
		name string
		make func(out *strings.Builder) Printer
		want string
	}{
		{name: "bytes", make: func(out *strings.Builder) Printer { return Bytes(out, 8) }, want: "[0 0 0 0 0 0 0 44]"},
		{name: "chars", make: func(out *strings.Builder) Printer { return Chars(out, 8) }, want: ","},
		{name: "decimal", make: func(out *strings.Builder) Printer { return Decimal(out, 8) }, want: "44"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &strings.Builder{}

			printed, err := tc.make(out).Print(value)
			if err != nil {
				t.Fatalf("Print: %v", err)
			}

			if got := strings.TrimSuffix(out.String(), "\n"); got != tc.want {
				t.Errorf("wrote %q, want %q", got, tc.want)
			}
			// A print answers with the value it was given: it reads a tape, it does not
			// change one. Answering with the formatted text would answer with something that
			// is not a tape.
			if string(printed) != string(value) {
				t.Errorf("answered %v, want the value it was given", printed)
			}
		})
	}
}

// failing is a writer that cannot be written to, which is what a closed pipe is.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

// The error comes back. It used to be dropped — the write was "_, _ =" — so a program whose
// output went nowhere carried on as if it had been heard.
func TestAWriteThatFailsIsAnswered(t *testing.T) {
	for _, p := range []Printer{Bytes(failing{}, 8), Chars(failing{}, 8), Decimal(failing{}, 8)} {
		printed, err := p.Print(tape(1))

		if err == nil {
			t.Error("a broken pipe was reported as a successful print")
		}
		if printed != nil {
			t.Errorf("answered %v for a print that failed", printed)
		}
	}
}
