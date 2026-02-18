package byteutil

import "testing"

func TestIsNothing(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{name: "empty", b: []byte{}, want: true},
		{name: "8 bytes", b: []byte{0, 0, 0, 0, 0, 0, 0, 0}, want: false},
		{name: "8 bytes with padding", b: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}, want: false},
		{name: "not empty", b: []byte{1, 2, 3}, want: false},
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
