package byteutil

import "testing"

func TestZeroFilledLength(t *testing.T) {
	cases := []struct {
		Expected int
		Value    []byte
	}{
		{2, append(FromUint64(1), FromUint64(2)...)},
		{2, append(FromUint64(10), FromUint64(20)...)},
		{2, append(FromUint64(100), FromUint64(200)...)},
		{2, append(FromUint64(1000), FromUint64(2000)...)},
	}
	for _, c := range cases {
		got := NonZeroFilledLength(c.Value)
		if expected := c.Expected; got != expected {
			t.Errorf("Unexpected length: got %v, expected: %v", got, expected)
		}
	}
}
