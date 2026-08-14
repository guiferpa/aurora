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
// is only a bug because of the number it produces.
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

// Multiplication and division group to the left, like addition and subtraction. They used to
// group to the right, so a chain answered for a different expression than the one written:
// 20 / 5 / 2 read as 20 / (5 / 2). Multiplication alone hid it — it is associative, and only
// a chain mixing it with division tells the two groupings apart.
func TestArithmeticAssociativity(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "printd 20 / 5 / 2;", want: "2"},
		{source: "printd 4 / 2 * 6;", want: "12"},
		{source: "printd 2 * 3 / 6;", want: "1"},
		{source: "printd 24 / 2 / 3 / 2;", want: "2"},
		{source: "printd 8 / 4 * 2;", want: "4"},
		{source: "printd 20 - 5 - 3;", want: "12"},
		{source: "printd 10 - 3 + 5;", want: "12"},
		{source: "printd 2 + 3 * 4;", want: "14"},
		{source: "printd (2 + 3) * 4;", want: "20"},
		// Exponentiation is the one that groups to the right, as it should: 2 ^ (3 ^ 2).
		{source: "printd 2 ^ 3 ^ 2;", want: "512"},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := output(t, tc.source, 0)[0]; got != tc.want {
				t.Errorf("%s printed %s, want %s", tc.source, got, tc.want)
			}
		})
	}
}
