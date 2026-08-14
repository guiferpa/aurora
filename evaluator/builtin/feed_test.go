package builtin

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// feed(n) reads the vector of values applied to a scope. The scope never learns how many
// it received, so the index wraps around the length: the read always answers with a tape
// and never fails. It used to return nil for anything out of range, which is not a value
// at all — "print feed(0)" with nothing applied printed an empty array.
func TestFeedFunction(t *testing.T) {
	feed := map[uint64][]byte{
		0: byteutil.PaddingTape([]byte{10}, 8),
		1: byteutil.PaddingTape([]byte{20}, 8),
	}

	cases := []struct {
		name  string
		index uint64
		want  byte
	}{
		{name: "first", index: 0, want: 10},
		{name: "second", index: 1, want: 20},
		{name: "one past the end wraps to the first", index: 2, want: 10},
		{name: "and keeps wrapping", index: 3, want: 20},
		{name: "far past the end", index: 100, want: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FeedFunction(feed, tc.index, 8)
			want := byteutil.PaddingTape([]byte{tc.want}, 8)
			if !bytes.Equal(got, want) {
				t.Errorf("FeedFunction(index %d) = %v, want %v", tc.index, got, want)
			}
		})
	}
}

func TestFeedFunctionWithNothingApplied(t *testing.T) {
	for _, size := range []int{1, 8, 32} {
		for _, feed := range []map[uint64][]byte{nil, {}} {
			got := FeedFunction(feed, 0, size)
			if want := byteutil.FalseTape(size); !bytes.Equal(got, want) {
				t.Errorf("size %d: got %v, want a tape of zeros", size, got)
			}
		}
	}
}

// Values arrive as 32-byte ABI words from the command line, and come out as tapes.
func TestFeedFunctionNarrowsToTheTape(t *testing.T) {
	wide := make([]byte, 32)
	wide[31] = 7
	feed := map[uint64][]byte{0: wide}

	for _, size := range []int{1, 2, 8} {
		got := FeedFunction(feed, 0, size)
		if len(got) != size {
			t.Errorf("size %d: got %d bytes", size, len(got))
		}
		if got[len(got)-1] != 7 {
			t.Errorf("size %d: got %v, want it to end in 7", size, got)
		}
	}
}
