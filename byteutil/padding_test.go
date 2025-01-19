package byteutil

import (
	"bytes"
	"testing"
)

func TestPadding64Bits(t *testing.T) {
	cases := []struct {
		Expected []byte
		Value    []byte
	}{
		{[]byte{0, 0, 0, 0, 0, 0, 0, 10}, []byte{10}},
		{[]byte{0, 0, 0, 0, 0, 0, 100, 100}, []byte{100, 100}},
		{[]byte{0, 0, 0, 0, 255, 255, 255, 0}, []byte{255, 255, 255, 0}},
	}
	for _, c := range cases {
		got := Padding64Bits(c.Value)
		if expected := c.Expected; bytes.Compare(got, expected) != 0 {
			t.Errorf("Unexpected padding: got %v, expected: %v", got, expected)
		}
	}
}
