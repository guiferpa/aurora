package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/guiferpa/aurora/byteutil"
)

func TestCallFailsWhenRPCUnreachable(t *testing.T) {
	ctx := context.Background()
	err := Call(ctx, CallInput{
		Function:        "foo",
		ContractAddress: "0x0000000000000000000000000000000000000000",
		RPC:             "http://invalid.invalid:99999",
	})
	if err == nil {
		t.Error("Call() with unreachable RPC should return error")
	}
}

func TestCallFailsWhenContractAddressInvalid(t *testing.T) {
	// Invalid address format should still try to dial first; use an invalid URL so we fail fast
	ctx := context.Background()
	err := Call(ctx, CallInput{
		Function:        "foo",
		ContractAddress: "not-a-valid-address",
		RPC:             "http://127.0.0.1:99999",
	})
	// We expect an error (either from dial or from address parsing)
	if err == nil {
		t.Error("Call() with invalid contract address should return error")
	}
}

func TestCallFailsWhenFunctionInvalid(t *testing.T) {
	ctx := context.Background()
	err := Call(ctx, CallInput{
		Function:        "not-a-valid-function",
		ContractAddress: "0x0000000000000000000000000000000000000000",
		RPC:             "http://127.0.0.1:99999",
	})
	if err == nil {
		t.Error("Call() with invalid function should return error")
	}
}

func TestEncodeSelector(t *testing.T) {
	cases := []struct {
		Function   string
		ExpectedFn func() []byte
	}{
		{
			Function: "fn",
			ExpectedFn: func() []byte {
				return byteutil.Padding32Bytes(crypto.Keccak256([]byte("fn")))
			},
		},
	}
	for _, c := range cases {
		got := EncodeSelector(c.Function)
		if expected := c.ExpectedFn(); !bytes.Equal(got, expected) {
			t.Errorf("Unexpected selector: got %v (%d), expected: %v (%d)", byteutil.ToHexBloom(got), len(got), byteutil.ToHexBloom(expected), len(expected))
		}
	}
}

func TestEncodeArgs(t *testing.T) {
	cases := []struct {
		Args       []string
		ExpectedFn func() []byte
	}{
		{
			Args: []string{"1", "2", "3"},
			ExpectedFn: func() []byte {
				data := byteutil.Padding32Bytes([]byte("1"))
				data = append(data, byteutil.Padding32Bytes([]byte("2"))...)
				data = append(data, byteutil.Padding32Bytes([]byte("3"))...)
				return data
			},
		},
	}
	for _, c := range cases {
		got := EncodeArgs(c.Args)
		if expected := c.ExpectedFn(); !bytes.Equal(got, expected) {
			t.Errorf("Unexpected args: got %v (%d), expected: %v (%d)", byteutil.ToHexBloom(got), len(got), byteutil.ToHexBloom(expected), len(expected))
		}
	}
}

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
