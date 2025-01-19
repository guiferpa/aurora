package byteutil

import (
	"bytes"
	"testing"
)

func TestFromUint64(t *testing.T) {
	cases := []struct {
		Expected []byte
		Value    uint64
	}{
		{[]byte{0, 0, 0, 0, 0, 0, 0, 10}, 10},
		{[]byte{0, 0, 0, 0, 0, 0, 0, 100}, 100},
		{[]byte{0, 0, 0, 0, 0, 0, 3, 232}, 1000},
		{[]byte{0, 0, 0, 0, 0, 0, 39, 16}, 10000},
	}
	for _, c := range cases {
		got := FromUint64(c.Value)
		if expected := c.Expected; !bytes.Equal(got, expected) {
			t.Errorf("Unexpected byte slice: got %v, expected: %v", got, expected)
		}
	}
}

func TestToUint64(t *testing.T) {
	cases := []struct {
		Expected uint64
		Value    []byte
	}{
		{10, []byte{0, 0, 0, 0, 0, 0, 0, 10}},
		{100, []byte{0, 0, 0, 0, 0, 0, 0, 100}},
		{1000, []byte{0, 0, 0, 0, 0, 0, 3, 232}},
		{10000, []byte{0, 0, 0, 0, 0, 0, 39, 16}},
	}
	for _, c := range cases {
		got := ToUint64(c.Value)
		if expected := c.Expected; got != expected {
			t.Errorf("Unexpected uint64: got %v, expected: %v", got, expected)
		}
	}
}
