package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeploy_failsWhenBytecodeMissing(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	// Create a valid-looking key file so we fail on bytecode read, not key read
	if err := os.WriteFile(keyPath, []byte("0000000000000000000000000000000000000000000000000000000000000001"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err := Deploy(ctx, DeployInput{
		BytecodePath:   filepath.Join(dir, "nonexistent.bin"),
		RPCURL:        "http://127.0.0.1:8545",
		PrivateKeyPath: keyPath,
	})
	if err == nil {
		t.Error("Deploy() with missing bytecode file should return error")
	}
}

func TestDeploy_failsWhenPrivateKeyFileMissing(t *testing.T) {
	dir := t.TempDir()
	bytecodePath := filepath.Join(dir, "contract.bin")
	if err := os.WriteFile(bytecodePath, []byte{0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err := Deploy(ctx, DeployInput{
		BytecodePath:   bytecodePath,
		RPCURL:        "http://127.0.0.1:8545",
		PrivateKeyPath: filepath.Join(dir, "missing.key"),
	})
	if err == nil {
		t.Error("Deploy() with missing private key file should return error")
	}
}
