package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunExecutesAndWritesToStdout(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	source := "ident x = 1 + 10;\nprint x;\n"
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	ctx := context.Background()
	err := Run(ctx, RunInput{
		Source:  entry,
		Loggers: nil,
		Stdin:   nil,
		Stdout:  &stdout,
		Player:  nil,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("Run() should produce some stdout output")
	}
}

func TestRunFailsWhenSourceMissing(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	err := Run(ctx, RunInput{
		Source: filepath.Join(dir, "nonexistent.ar"),
		Stdout: os.Stdout,
	})
	if err == nil {
		t.Error("Run() with missing source should return error")
	}
}

func TestRunFailsWhenSourceInvalid(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "bad.ar")
	if err := os.WriteFile(entry, []byte("invalid {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	ctx := context.Background()
	err := Run(ctx, RunInput{
		Source: entry,
		Stdout: &stdout,
	})
	if err == nil {
		t.Error("Run() with invalid source should return error")
	}
}
