package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Compiling is no longer a thing of its own: a command compiles as part of what it does, so
// what a source does to the compiler is asked through a command. These used to call Compile,
// which was a step nobody ran on its own.

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

func TestASourceBecomesInstructions(t *testing.T) {
	dir := t.TempDir()
	// Nothing a chain cannot carry, so nothing to warn about: a print would be warned about
	// by the backend, which is a different thing from the compiler having something to say.
	source := writeFile(t, dir, "main.ar", "ident a = 1;\nident b = a + 1;\n")
	warnings := &strings.Builder{}

	report, err := newSession(t, sessionOpts{warnings: warnings}).
		Build(t.Context(), source, filepath.Join(dir, "bin", "main"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if report.Instructions == 0 {
		t.Error("the source produced no instructions")
	}
	if got := warnings.String(); got != "" {
		t.Errorf("the compiler said %q about a program it has nothing to say about", got)
	}
}

// The unit of compilation is the file. A namespace layer used to compile every .ar file
// in the directory, so two independent programs sharing a folder collided on their
// identifiers — which is what made "aurora run" fail on the examples directory.
func TestANeighbouringFileIsNotCompiledIn(t *testing.T) {
	dir := t.TempDir()
	source := writeFile(t, dir, "main.ar", "ident a = 1;\nprintb a;\n")
	writeFile(t, dir, "other.ar", "ident a = 2;\nprintb a;\n")
	writeFile(t, dir, "third.ar", "ident a = 3;\n")

	if err := newSession(t, sessionOpts{}).Run(t.Context(), source); err != nil {
		t.Fatalf("a neighbour must not be compiled in: %v", err)
	}
}

func TestASourceThatDoesNotCompile(t *testing.T) {
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

			err := newSession(t, sessionOpts{tapeSize: tapeSize}).Run(t.Context(), source)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestASourceThatIsNotThere(t *testing.T) {
	err := newSession(t, sessionOpts{}).Run(t.Context(), filepath.Join(t.TempDir(), "nope.ar"))
	if err == nil {
		t.Error("expected an error for a missing file")
	}
}

// assert is only accepted in *.test.ar, and the path is what says which one this is.
func TestAssertBelongsToATestFile(t *testing.T) {
	dir := t.TempDir()
	const source = "assert(1 equals 1, \"ok\");\n"

	if err := newSession(t, sessionOpts{}).Run(t.Context(), writeFile(t, dir, "checks.test.ar", source)); err != nil {
		t.Errorf("assert should be accepted in a .test.ar file: %v", err)
	}
	if err := newSession(t, sessionOpts{}).Run(t.Context(), writeFile(t, dir, "checks.ar", source)); err == nil {
		t.Error("assert should be rejected outside .test.ar")
	}
}

// A short call is told about where it was written, in the form an editor follows.
//
// The whole point of the warning is that the answer is silent: reading past what was applied
// gives a tape of zeros, so nothing at runtime says the value never arrived. It has to reach
// the person, with a place to look — which is the part no test of the emitter alone can show,
// since it crosses the loader, the session and the writer to get there.
func TestAShortCallIsReportedWithWhereToLook(t *testing.T) {
	dir := t.TempDir()
	source := writeFile(t, dir, "main.ar", "ident sum = defer { feed(0) + feed(1); };\nprintb sum(5);\n")
	warnings := &strings.Builder{}

	if _, err := newSession(t, sessionOpts{warnings: warnings}).
		Build(t.Context(), source, filepath.Join(dir, "bin", "main")); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := warnings.String()
	for _, want := range []string{"main.ar:2:8:", "sum reads 2 positions", "feed(1)"} {
		if !strings.Contains(got, want) {
			t.Errorf("the compiler said %q, want it to mention %q", got, want)
		}
	}
}
