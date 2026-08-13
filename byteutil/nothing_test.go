package byteutil

import "testing"

func TestIsNothing(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		// Nothing is a tape of zeros, the same representation as the number zero.
		{name: "tape of zeros", b: []byte{0, 0, 0, 0, 0, 0, 0, 0}, want: true},
		{name: "one zero byte", b: []byte{0}, want: true},
		{name: "reel of zeros", b: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}, want: true},
		{name: "has a non-zero byte", b: []byte{0, 0, 1}, want: false},
		{name: "not empty", b: []byte{1, 2, 3}, want: false},
		{name: "empty is not a value", b: []byte{}, want: false},
		{name: "nil", b: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNothing(tt.b); got != tt.want {
				t.Errorf("IsNothing(%v) = %v, want %v", tt.b, got, tt.want)
			}
		})
	}
}
