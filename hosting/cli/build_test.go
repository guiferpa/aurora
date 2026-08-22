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
	_, err := newSession(t, sessionOpts{}).Build(ctx, entry, out)
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
	_, err := newSession(t, sessionOpts{}).Build(ctx, filepath.Join(dir, "nonexistent.ar"), out)
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
	_, err := newSession(t, sessionOpts{}).Build(ctx, entry, out)
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
	report, err := newSession(t, sessionOpts{stdout: stdout}).Build(t.Context(), entry, out)
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
	report, err := newSession(t, sessionOpts{tapeSize: 4, stdout: stdout}).Build(t.Context(), entry, filepath.Join(dir, "out.bin"))
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
	if _, err := newSession(t, sessionOpts{stdout: nil}).Build(t.Context(), entry, out); err != nil {
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

// A binary that quietly does less than the source said is the worst thing a build can hand
// someone. What the backend cannot carry yet is said out loud, at the moment it is built.
func TestBuildSaysWhatTheBackendDoesNotCarry(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "a log",
			source: "ident show = defer { printd feed(0); };\n",
			want:   "printd writes a log",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "main.ar")
			if err := os.WriteFile(source, []byte(tc.source), 0o644); err != nil {
				t.Fatal(err)
			}

			warnings := &strings.Builder{}
			if _, err := newSession(t, sessionOpts{warnings: warnings}).Build(t.Context(), source, filepath.Join(dir, "out.bin")); err != nil {
				t.Fatalf("Build: %v", err)
			}

			if got := warnings.String(); !strings.Contains(got, tc.want) {
				t.Errorf("the build said %q, want it to say %q", got, tc.want)
			}
		})
	}
}

// A program the backend writes whole is built without a word about it.
func TestBuildSaysNothingAboutWhatItCarries(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.ar")
	if err := os.WriteFile(source, []byte("ident add = defer { feed(0) + feed(1); };\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	warnings := &strings.Builder{}
	if _, err := newSession(t, sessionOpts{warnings: warnings}).Build(t.Context(), source, filepath.Join(dir, "out.bin")); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := warnings.String(); got != "" {
		t.Errorf("the build said %q about a program it carries whole", got)
	}
}
