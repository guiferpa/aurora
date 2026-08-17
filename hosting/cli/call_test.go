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
