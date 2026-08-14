package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

func TestCompile(t *testing.T) {
	dir := t.TempDir()
	source := writeFile(t, dir, "main.ar", "ident a = 1;\nprintb a + 1;\n")

	program, err := Compile(source, 0, nil)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(program.Instructions) == 0 {
		t.Error("expected instructions")
	}
	if len(program.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", program.Warnings)
	}
}

// The unit of compilation is the file. A namespace layer used to compile every .ar file
// in the directory, so two independent programs sharing a folder collided on their
// identifiers — which is what made "aurora run" fail on the examples directory.
func TestCompileIgnoresNeighbouringFiles(t *testing.T) {
	dir := t.TempDir()
	source := writeFile(t, dir, "main.ar", "ident a = 1;\nprintb a;\n")
	writeFile(t, dir, "other.ar", "ident a = 2;\nprintb a;\n")
	writeFile(t, dir, "third.ar", "ident a = 3;\n")

	if _, err := Compile(source, 0, nil); err != nil {
		t.Fatalf("a neighbour must not be compiled in: %v", err)
	}
}

func TestCompileReportsErrors(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		source  string
		wantErr string
	}{
		{name: "lexer error", source: "ident a = @;\n", wantErr: "unexpected character"},
		{name: "parser error", source: "ident = 1;\n", wantErr: "unexpected token"},
		{name: "literal too wide for the tape", source: "ident a = 300;\n", wantErr: "does not fit"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tapeSize := 0
			if strings.Contains(tc.wantErr, "does not fit") {
				tapeSize = 1
			}
			source := writeFile(t, dir, strings.ReplaceAll(tc.name, " ", "_")+".ar", tc.source)
			_, err := Compile(source, tapeSize, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestCompileFailsWhenSourceMissing(t *testing.T) {
	if _, err := Compile(filepath.Join(t.TempDir(), "nope.ar"), 0, nil); err == nil {
		t.Error("expected an error for a missing file")
	}
}

// assert is only accepted in *.test.ar, and Compile passes the path through for that.
func TestCompileHonoursTestFileRule(t *testing.T) {
	dir := t.TempDir()
	const source = "assert(1 equals 1, \"ok\");\n"

	if _, err := Compile(writeFile(t, dir, "checks.test.ar", source), 0, nil); err != nil {
		t.Errorf("assert should be accepted in a .test.ar file: %v", err)
	}
	if _, err := Compile(writeFile(t, dir, "checks.ar", source), 0, nil); err == nil {
		t.Error("assert should be rejected outside .test.ar")
	}
}
