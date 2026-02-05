package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const minimalAR = "ident x = 1 + 2;\nprint x;\n"

func TestBuild_producesOutputFile(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(entry, []byte(minimalAR), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", "main.bin")
	ctx := context.Background()
	err := Build(ctx, BuildInput{
		Entrypoint: entry,
		OutputPath: out,
		Loggers:    nil,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestBuild_failsWhenEntrypointMissing(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.bin")
	ctx := context.Background()
	err := Build(ctx, BuildInput{
		Entrypoint: filepath.Join(dir, "nonexistent.ar"),
		OutputPath: out,
	})
	if err == nil {
		t.Error("Build() with missing entrypoint should return error")
	}
}

func TestBuild_failsWhenSourceInvalid(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "bad.ar")
	if err := os.WriteFile(entry, []byte("invalid syntax {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.bin")
	ctx := context.Background()
	err := Build(ctx, BuildInput{Entrypoint: entry, OutputPath: out})
	if err == nil {
		t.Error("Build() with invalid source should return error")
	}
}
