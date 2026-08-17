package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalAR = "ident x = 1 + 2;\nprintb x;\n"

func TestBuild_producesOutputFile(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(entry, []byte(minimalAR), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", "main.bin")
	ctx := context.Background()
	_, err := Build(ctx, BuildInput{
		Source:     entry,
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

func TestBuild_failsWhenSourceMissing(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.bin")
	ctx := context.Background()
	_, err := Build(ctx, BuildInput{
		Source:     filepath.Join(dir, "nonexistent.ar"),
		OutputPath: out,
	})
	if err == nil {
		t.Error("Build() with missing source should return error")
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
	_, err := Build(ctx, BuildInput{Source: entry, OutputPath: out})
	if err == nil {
		t.Error("Build() with invalid source should return error")
	}
}

// A build says what it produced. The numbers have to describe the binary that is actually
// on disk, or the report is worse than the silence it replaced.
func TestBuildReportsWhatItProduced(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(entry, []byte(minimalAR), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bin", "main")

	stdout := &strings.Builder{}
	report, err := Build(t.Context(), BuildInput{Source: entry, OutputPath: out, Stdout: stdout})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if int64(report.Bytes) != info.Size() {
		t.Errorf("report says %d bytes, the file has %d", report.Bytes, info.Size())
	}
	if report.Instructions == 0 {
		t.Error("a program that does something has instructions")
	}
	if report.TapeSize != 8 {
		t.Errorf("tape size = %d, want the default 8", report.TapeSize)
	}
	if report.Source != entry || report.Binary != out {
		t.Errorf("report = %s → %s, want %s → %s", report.Source, report.Binary, entry, out)
	}

	written := stdout.String()
	for _, want := range []string{"main.ar", "main", "instructions", "bytes", "8-byte tapes"} {
		if !strings.Contains(written, want) {
			t.Errorf("report %q does not mention %q", written, want)
		}
	}
}

// The tape width is a compile-time decision that the binary carries with it, so the report
// has to name the one that was used rather than the default.
func TestBuildReportsTheTapeSizeItUsed(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(entry, []byte("printb 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := &strings.Builder{}
	report, err := Build(t.Context(), BuildInput{
		Source:     entry,
		OutputPath: filepath.Join(dir, "out.bin"),
		TapeSize:   4,
		Stdout:     stdout,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if report.TapeSize != 4 {
		t.Errorf("tape size = %d, want 4", report.TapeSize)
	}
	if !strings.Contains(stdout.String(), "4-byte tapes") {
		t.Errorf("report %q does not name the tape width", stdout.String())
	}
}

// Nowhere to write is not a reason to fail, and it is how every existing caller builds.
func TestBuildWithoutAReportStillBuilds(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(entry, []byte(minimalAR), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.bin")
	if _, err := Build(t.Context(), BuildInput{Source: entry, OutputPath: out, Stdout: nil}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}

func TestPlural(t *testing.T) {
	cases := []struct {
		count int
		want  string
	}{
		{count: 0, want: "0 bytes"},
		{count: 1, want: "1 byte"},
		{count: 2, want: "2 bytes"},
	}
	for _, tc := range cases {
		if got := plural(tc.count, "byte"); got != tc.want {
			t.Errorf("plural(%d) = %q, want %q", tc.count, got, tc.want)
		}
	}
}
