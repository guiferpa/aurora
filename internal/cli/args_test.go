package cli

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		Args       []string
		ExpectedFn func() []byte
	}{
		{
			Args: []string{"true", "false"},
			ExpectedFn: func() []byte {
				// bool: 1 and 0 right-padded to 32 bytes
				tr := byteutil.Padding32Bytes([]byte{1})
				fa := byteutil.Padding32Bytes([]byte{0})
				return append(tr, fa...)
			},
		},
		{
			Args: []string{"42", "0x2a"},
			ExpectedFn: func() []byte {
				// number 42 as uint256 big-endian (decimal and hex)
				word := make([]byte, 32)
				word[31] = 42
				return append(word, word...)
			},
		},
		{
			Args: []string{`""`},
			ExpectedFn: func() []byte {
				return byteutil.Padding32Bytes([]byte{}) // empty string
			},
		},
		{
			Args: []string{`"hello"`},
			ExpectedFn: func() []byte {
				return byteutil.Padding32Bytes([]byte("hello"))
			},
		},
	}
	for _, c := range cases {
		got := ParseArgs(c.Args)
		expected := c.ExpectedFn()
		if !bytes.Equal(got, expected) {
			t.Errorf("ParseArgs(%q): got %v (%d), expected: %v (%d)", c.Args, byteutil.ToHexBloom(got), len(got), byteutil.ToHexBloom(expected), len(expected))
		}
	}
}
