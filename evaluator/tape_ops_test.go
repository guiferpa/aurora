package evaluator

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// A tape is a shift register: pull moves it left with the item entering at the right, push
// moves it right with the item entering at the left, and whatever reaches the far end is
// discarded. The cases below are the ones written in docs/language-design.md.
func TestTapeOperations(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   []byte
	}{
		{
			name:   "a tape literal is built by pulling each value",
			source: "[1, 2, 3];",
			want:   []byte{0, 0, 0, 0, 0, 1, 2, 3},
		},
		{
			name:   "an empty tape literal is zeros",
			source: "[];",
			want:   []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:   "pull appends at the right",
			source: "ident a = [1, 2, 3];\npull a 4;",
			want:   []byte{0, 0, 0, 0, 1, 2, 3, 4},
		},
		{
			name:   "push prepends at the left, dropping the far end",
			source: "ident b = [1, 2, 3, 4];\npush b 5;",
			want:   []byte{5, 0, 0, 0, 0, 1, 2, 3},
		},
		{
			name:   "pull concatenates the significant bytes of a tape",
			source: "ident a = [1, 2];\npull a [3, 4];",
			want:   []byte{0, 0, 0, 0, 1, 2, 3, 4},
		},
		{
			name:   "head keeps the first significant bytes",
			source: "ident a = [1, 2, 3, 4, 5];\nhead a 2;",
			want:   []byte{0, 0, 0, 0, 0, 0, 1, 2},
		},
		{
			name:   "tail drops the first significant bytes",
			source: "ident a = [1, 2, 3, 4, 5];\ntail a 2;",
			want:   []byte{0, 0, 0, 0, 0, 3, 4, 5},
		},
		{
			name:   "head on a full tape",
			source: "ident a = [1, 2, 3, 4, 5, 6, 7, 8];\nhead a 2;",
			want:   []byte{0, 0, 0, 0, 0, 0, 1, 2},
		},
		{
			name:   "tail on a full tape",
			source: "ident a = [1, 2, 3, 4, 5, 6, 7, 8];\ntail a 2;",
			want:   []byte{0, 0, 3, 4, 5, 6, 7, 8},
		},
		{
			// The index is taken modulo the width, so it can never be out of bounds.
			name:   "head with the width as index keeps nothing",
			source: "ident a = [1, 2, 3, 4, 5, 6, 7, 8];\nhead a 8;",
			want:   []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:   "an index past the width wraps",
			source: "ident a = [1, 2, 3, 4, 5, 6, 7, 8];\ntail a 18;",
			want:   []byte{0, 0, 3, 4, 5, 6, 7, 8},
		},
		{
			name:   "pulling into a full tape discards on the left",
			source: "ident a = [1, 2, 3, 4, 5, 6, 7, 8];\npull a 9;",
			want:   []byte{2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			// Everything is a tape, so the operations work on any value, not only literals.
			name:   "a number is a tape too",
			source: "ident a = 1;\npull a 2;",
			want:   []byte{0, 0, 0, 0, 0, 0, 1, 2},
		},
		{
			name:   "pushing onto a number",
			source: "ident a = 1;\npush a 2;",
			want:   []byte{2, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runWithTapeSize(t, tc.source, byteutil.DefaultTapeSize)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("%q = %v, want %v", tc.source, got, tc.want)
			}
		})
	}
}

// The tape width drives the operations, so a narrow tape shifts sooner.
func TestTapeOperationsFollowTheTapeSize(t *testing.T) {
	cases := []struct {
		name   string
		source string
		size   int
		want   []byte
	}{
		{name: "two bytes hold two values", source: "[1, 2];", size: 2, want: []byte{1, 2}},
		{name: "pulling into a full two-byte tape", source: "ident a = [1, 2];\npull a 3;", size: 2, want: []byte{2, 3}},
		{name: "pushing into a full two-byte tape", source: "ident a = [1, 2];\npush a 9;", size: 2, want: []byte{9, 1}},
		{name: "one byte holds one value", source: "[7];", size: 1, want: []byte{7}},
		{name: "pulling replaces the only byte", source: "ident a = [7];\npull a 8;", size: 1, want: []byte{8}},
		// With one byte every index is zero: head keeps nothing, tail keeps everything.
		{name: "head with one-byte tapes", source: "ident a = [7];\nhead a 1;", size: 1, want: []byte{0}},
		{name: "tail with one-byte tapes", source: "ident a = [7];\ntail a 1;", size: 1, want: []byte{7}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runWithTapeSize(t, tc.source, tc.size)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("%q with %d-byte tapes = %v, want %v", tc.source, tc.size, got, tc.want)
			}
		})
	}
}

// The value of a tape operation is an ordinary tape, so it can be bound and used.
func TestTapeOperationsProduceUsableValues(t *testing.T) {
	got := runWithTapeSize(t, "ident a = [1, 2];\nident b = pull a 3;\nb + 1;", byteutil.DefaultTapeSize)
	want := byteutil.PaddingTape([]byte{1, 2, 4}, byteutil.DefaultTapeSize) // 0x010203 + 1
	if !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
