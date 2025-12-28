package byteutil

import "testing"

func TestToHexBloom(t *testing.T) {
	cases := []struct {
		Input    []byte
		Expected string
	}{
		{[]byte{0x00, 0x01, 0x02, 0x03}, "00 01 02 03"},
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, "00 01 02 03 04 05 06 07"},
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}, "00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F"},
		{[]byte{0x00, 10, 0x00}, "00 0A 00"},
	}
	for _, c := range cases {
		got := ToHexBloom(c.Input)
		if got != c.Expected {
			t.Errorf("unexpected hex bloom, got: %s, expected: %s, input: %v.", got, c.Expected, c.Input)
		}
	}
}
