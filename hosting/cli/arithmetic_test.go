package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// output runs source through the whole pipeline and returns what it printed, one line per
// print. Arithmetic is worth checking at the answer rather than at the tree: a wrong shape
// is only a bug because of the number it produces. Shared with the precedence cases.
func output(t *testing.T, source string, tapeSize int) []string {
	t.Helper()

	entry := filepath.Join(t.TempDir(), "main.ar")
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Run(t.Context(), RunInput{Source: entry, Stdout: &stdout, TapeSize: tapeSize}); err != nil {
		t.Fatalf("running %q: %v", source, err)
	}
	return strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
}

// A tape is unsigned, so negating wraps: -x is whatever 0 - x is. The operator used to be
// dropped between the parser and the emitter, which made -5 print 5 and 10 + -5 print 15.
func TestNegation(t *testing.T) {
	const max = "18446744073709551611" // 2^64 - 5, which is what 0 - 5 gives

	cases := []struct {
		source string
		want   string
	}{
		{source: "printd -5;", want: max},
		{source: "printd 0 - 5;", want: max},
		{source: "ident a = 5;\nprintd -a;", want: max},
		{source: "printd -(2 + 3);", want: max},
		{source: "printd 10 + -5;", want: "5"},
		{source: "printd 10 - -5;", want: "15"},
		{source: "ident a = 5;\nprintd -a * 2;", want: "18446744073709551606"}, // 2^64 - 10
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := output(t, tc.source, 0)[0]; got != tc.want {
				t.Errorf("%s printed %s, want %s", tc.source, got, tc.want)
			}
		})
	}
}

// Negation wraps at whatever the tape holds, not at 64 bits.
func TestNegationAcrossTapeSizes(t *testing.T) {
	cases := []struct {
		tapeSize int
		want     string
	}{
		{tapeSize: 1, want: "251"},   // 2^8 - 5
		{tapeSize: 2, want: "65531"}, // 2^16 - 5
		{tapeSize: 8, want: "18446744073709551611"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := output(t, "printd -5;", tc.tapeSize)[0]
			if got != tc.want {
				t.Errorf("%d-byte tapes: -5 printed %s, want %s", tc.tapeSize, got, tc.want)
			}
			// And it is the same value the subtraction gives, which is the whole claim.
			if bySubtraction := output(t, "printd 0 - 5;", tc.tapeSize)[0]; bySubtraction != got {
				t.Errorf("%d-byte tapes: -5 is %s but 0 - 5 is %s", tc.tapeSize, got, bySubtraction)
			}
		})
	}
}
