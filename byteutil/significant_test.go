package byteutil

import (
	"bytes"
	"testing"
)

func TestExtractSignificantBytes(t *testing.T) {
	cases := []struct {
		Name     string
		Input    []byte
		Expected []byte
	}{
		{
			Name:     "empty slice",
			Input:    []byte{},
			Expected: []byte{0},
		},
		{
			Name:     "single zero",
			Input:    []byte{0},
			Expected: []byte{0},
		},
		{
			Name:     "all zeros",
			Input:    []byte{0, 0, 0, 0},
			Expected: []byte{0},
		},
		{
			Name:     "all zeros 8 bytes",
			Input:    []byte{0, 0, 0, 0, 0, 0, 0, 0},
			Expected: []byte{0},
		},
		{
			Name:     "single non-zero at start",
			Input:    []byte{1},
			Expected: []byte{1},
		},
		{
			Name:     "single non-zero at end",
			Input:    []byte{0, 0, 0, 0, 0, 0, 0, 1},
			Expected: []byte{1},
		},
		{
			Name:     "multiple bytes, no leading zeros",
			Input:    []byte{1, 2, 3},
			Expected: []byte{1, 2, 3},
		},
		{
			Name:     "multiple bytes with leading zeros",
			Input:    []byte{0, 0, 0, 1, 2, 3},
			Expected: []byte{1, 2, 3},
		},
		{
			Name:     "8 bytes with leading zeros",
			Input:    []byte{0, 0, 0, 0, 0, 1, 2, 3},
			Expected: []byte{1, 2, 3},
		},
		{
			Name:     "mixed zeros and non-zeros",
			Input:    []byte{0, 0, 5, 0, 7, 8},
			Expected: []byte{5, 0, 7, 8},
		},
		{
			Name:     "large number representation",
			Input:    []byte{0, 0, 0, 0, 0, 1, 44}, // 300 in decimal
			Expected: []byte{1, 44},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := ExtractSignificantBytes(c.Input)
			if !bytes.Equal(result, c.Expected) {
				t.Errorf("ExtractSignificantBytes(%v) = %v, expected %v", c.Input, result, c.Expected)
			}
		})
	}
}

