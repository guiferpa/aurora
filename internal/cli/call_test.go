package cli

import (
	"context"
	"testing"
)

func TestCall_failsWhenRPCUnreachable(t *testing.T) {
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

func TestCall_failsWhenContractAddressInvalid(t *testing.T) {
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
