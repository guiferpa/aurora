package byteutil

import "testing"

func TestToBoolean(t *testing.T) {
	cases := []struct {
		Expected bool
		Value    []byte
	}{
		{false, []byte{}},
		{false, []byte{0}},
		{false, []byte{0, 0}},
		{false, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{false, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{true, []byte{0, 0, 0, 0, 0, 0, 0, 0, 1}}, // Truncated to last 8 bytes: [0, 0, 0, 0, 0, 0, 0, 1] = true
		{false, False},
		{true, []byte{1}},
		{true, []byte{1, 1}},
		{true, []byte{0, 255, 255}},
		{true, []byte{0, 0, 0, 0, 0, 0, 0, 1}},
		{true, True},
	}
	for _, c := range cases {
		got := ToBoolean(c.Value)
		if expected := c.Expected; got != expected {
			t.Errorf("Unexpected bool: got %v, expected: %v", got, expected)
		}
	}
}
