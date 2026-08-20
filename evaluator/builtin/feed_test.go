package builtin

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// tape builds a tape of the given width holding a number.
func tape(size int, value uint64) []byte {
	return byteutil.PaddingTape(byteutil.FromUint64(value), size)
}

// feed(n) reads the vector of values applied to a scope. The scope never learns how many
// it received, so the index wraps around the length: the read always answers with a tape
// and never fails. It used to return nil for anything out of range, which is not a value
// at all — "printb feed(0)" with nothing applied printed an empty array.
func TestFeedFunction(t *testing.T) {
	feed := map[uint64][]byte{
		0: tape(8, 10),
		1: tape(8, 20),
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
			want := tape(8, uint64(tc.want))
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

// A value wider than a tape is a run of them — a reel, or a shape — and feed hands it over
// whole. It used to narrow every read to a single tape, which cut a shape down to its last
// field: `defer { (feed(0) as Point).x }` answered with y.
//
// The narrowing that command-line arguments need is not gone, it lives where those arguments
// enter: environ.NewEnviron turns each 32-byte ABI word into a tape.
func TestFeedFunctionHandsARunOverWhole(t *testing.T) {
	run := make([]byte, 0, 16)
	run = append(run, tape(8, 10)...)
	run = append(run, tape(8, 20)...)

	got := FeedFunction(map[uint64][]byte{0: run}, 0, 8)
	if !bytes.Equal(got, run) {
		t.Errorf("got %v, want the run whole: %v", got, run)
	}
}

// Anything narrower than a tape still comes back as one, so a read always answers with at
// least a whole value.
func TestFeedFunctionPadsWhatIsNarrowerThanATape(t *testing.T) {
	for _, size := range []int{2, 8, 32} {
		got := FeedFunction(map[uint64][]byte{0: {7}}, 0, size)
		if len(got) != size {
			t.Errorf("size %d: got %d bytes", size, len(got))
		}
		if got[len(got)-1] != 7 {
			t.Errorf("size %d: got %v, want it to end in 7", size, got)
		}
	}
}
