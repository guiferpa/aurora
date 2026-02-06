package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Valid 64-char hex key for tests (no 0x prefix).
const testPrivkeyHex = "0000000000000000000000000000000000000000000000000000000000000001"

func TestDeploy_failsWhenBytecodeMissing(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	_, _, _, err := Deploy(ctx, DeployInput{
		BinaryPath: filepath.Join(dir, "nonexistent.bin"),
		RPC:        "http://127.0.0.1:8545",
		Privkey:    testPrivkeyHex,
	})
	if err == nil {
		t.Error("Deploy() with missing bytecode file should return error")
	}
}

func TestDeploy_failsWhenPrivkeyInvalid(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "contract.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _, _, err := Deploy(ctx, DeployInput{
		BinaryPath: binaryPath,
		RPC:        "http://127.0.0.1:8545",
		Privkey:    "not-valid-hex",
	})
	if err == nil {
		t.Error("Deploy() with invalid privkey should return error")
	}
}
