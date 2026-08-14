package byteutil

import (
	"bytes"
	"testing"

	"github.com/holiman/uint256"
)

func TestTapeSizeNormalizes(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{name: "unset falls back to the default", in: 0, want: DefaultTapeSize},
		{name: "negative falls back to the default", in: -3, want: DefaultTapeSize},
		{name: "floor", in: 1, want: 1},
		{name: "in range", in: 4, want: 4},
		{name: "ceiling", in: 32, want: 32},
		{name: "above the ceiling is clamped", in: 64, want: MaxTapeSize},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TapeSize(tc.in); got != tc.want {
				t.Errorf("TapeSize(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPaddingTape(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		size int
		want []byte
	}{
		{name: "pads on the left", in: []byte{1}, size: 4, want: []byte{0, 0, 0, 1}},
		{name: "exact width is untouched", in: []byte{1, 2}, size: 2, want: []byte{1, 2}},
		{name: "keeps the last bytes when wider", in: []byte{1, 2, 3, 4}, size: 2, want: []byte{3, 4}},
		// A reel is a run of tapes: arithmetic on it uses the last one.
		{name: "reel narrows to its last tape", in: []byte{0, 104, 0, 105}, size: 2, want: []byte{0, 105}},
		{name: "empty becomes zeros", in: nil, size: 3, want: []byte{0, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PaddingTape(tc.in, tc.size); !bytes.Equal(got, tc.want) {
				t.Errorf("PaddingTape(%v, %d) = %v, want %v", tc.in, tc.size, got, tc.want)
			}
		})
	}
}

func TestUint256RoundTrip(t *testing.T) {
	for _, size := range []int{1, 2, 8, 32} {
		t.Run(sizeName(size), func(t *testing.T) {
			value := uint256.NewInt(42)
			tape := FromUint256(value, size)
			if len(tape) != size {
				t.Fatalf("tape has %d bytes, want %d", len(tape), size)
			}
			if got := ToUint256(tape, size); !got.Eq(value) {
				t.Errorf("round trip gave %s, want %s", got.Dec(), value.Dec())
			}
		})
	}
}

// A tape of N bytes holds values modulo 2^(8N): the width is what defines the wrap.
func TestFromUint256Wraps(t *testing.T) {
	cases := []struct {
		name  string
		value uint64
		size  int
		want  []byte
	}{
		{name: "256 wraps to 0 in one byte", value: 256, size: 1, want: []byte{0}},
		{name: "257 wraps to 1 in one byte", value: 257, size: 1, want: []byte{1}},
		{name: "255 fits in one byte", value: 255, size: 1, want: []byte{255}},
		{name: "65536 wraps to 0 in two bytes", value: 65536, size: 2, want: []byte{0, 0}},
		{name: "300 fits in two bytes", value: 300, size: 2, want: []byte{1, 44}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FromUint256(uint256.NewInt(tc.value), tc.size)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("FromUint256(%d, %d) = %v, want %v", tc.value, tc.size, got, tc.want)
			}
		})
	}
}

func TestFitsInTape(t *testing.T) {
	cases := []struct {
		name  string
		value uint64
		size  int
		want  bool
	}{
		{name: "255 in one byte", value: 255, size: 1, want: true},
		{name: "256 in one byte", value: 256, size: 1, want: false},
		{name: "65535 in two bytes", value: 65535, size: 2, want: true},
		{name: "65536 in two bytes", value: 65536, size: 2, want: false},
		{name: "max uint64 in eight bytes", value: ^uint64(0), size: 8, want: true},
		{name: "max uint64 in thirty-two bytes", value: ^uint64(0), size: 32, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FitsInTape(tc.value, tc.size); got != tc.want {
				t.Errorf("FitsInTape(%d, %d) = %v, want %v", tc.value, tc.size, got, tc.want)
			}
		})
	}
}

func TestMaxTapeValue(t *testing.T) {
	cases := []struct {
		size int
		want uint64
	}{
		{size: 1, want: 255},
		{size: 2, want: 65535},
		{size: 8, want: ^uint64(0)},
		{size: 32, want: ^uint64(0)},
	}
	for _, tc := range cases {
		t.Run(sizeName(tc.size), func(t *testing.T) {
			if got := MaxTapeValue(tc.size); got != tc.want {
				t.Errorf("MaxTapeValue(%d) = %d, want %d", tc.size, got, tc.want)
			}
		})
	}
}

// true and false are ordinary tapes: the only difference from a number is the value.
func TestBooleanTapesAreOrdinaryTapes(t *testing.T) {
	for _, size := range []int{1, 8, 32} {
		t.Run(sizeName(size), func(t *testing.T) {
			truth, lie := TrueTape(size), FalseTape(size)
			if len(truth) != size || len(lie) != size {
				t.Fatalf("boolean tapes have %d and %d bytes, want %d", len(truth), len(lie), size)
			}
			if !bytes.Equal(truth, FromUint256(uint256.NewInt(1), size)) {
				t.Errorf("true = %v, want the same tape as the number 1", truth)
			}
			if !ToBoolean(truth) || ToBoolean(lie) {
				t.Error("truthiness must follow the value, not the width")
			}
		})
	}
}

func sizeName(size int) string {
	return string(rune('0'+size/10)) + string(rune('0'+size%10)) + " bytes"
}
