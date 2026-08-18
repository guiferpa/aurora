package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compileWithLoggers compiles a source and returns what the traces wrote.
func compileWithLoggers(t *testing.T, source string, loggers []string) string {
	t.Helper()

	entry := filepath.Join(t.TempDir(), "main.ar")
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	traces := &strings.Builder{}
	if _, err := Compile(entry, 0, loggers, traces, nil); err != nil {
		t.Fatalf("compiling: %v", err)
	}
	return traces.String()
}

// -l is how someone sees what a phase produced, and each name shows a different thing. The
// phases print nothing themselves, so this is the only place the flag can be checked — and
// it is checked here rather than by running the binary and reading the terminal.
func TestLoggersShowWhatEachPhaseProduced(t *testing.T) {
	const source = "ident a = 1 + 2;\n"

	cases := []struct {
		logger string
		want   []string
	}{
		// The lexer: the tag of each token, and the bytes it matched.
		{logger: "lexer", want: []string{"IDENT", "NUMBER", "SUM", "SEMICOLON"}},
		// The parser: the tree, with every node named by its type.
		{logger: "parser", want: []string{"IdentLiteral", "BinaryExpression", "NumberLiteral"}},
		// The emitter: the instructions, by opcode.
		{logger: "emitter", want: []string{"OpSave", "OpAdd", "OpIdent"}},
	}

	for _, tc := range cases {
		t.Run(tc.logger, func(t *testing.T) {
			written := compileWithLoggers(t, source, []string{tc.logger})
			for _, want := range tc.want {
				if !strings.Contains(written, want) {
					t.Errorf("-l %s does not mention %s:\n%s", tc.logger, want, written)
				}
			}
		})
	}
}

// Asking for one phase shows that phase, and not the others: a flag that shows everything
// is a flag that shows nothing.
func TestOneLoggerDoesNotDragTheOthers(t *testing.T) {
	written := compileWithLoggers(t, "ident a = 1;\n", []string{"emitter"})

	if !strings.Contains(written, "OpSave") {
		t.Fatalf("-l emitter shows nothing:\n%s", written)
	}
	for _, other := range []string{"IdentLiteral", "SEMICOLON"} {
		if strings.Contains(written, other) {
			t.Errorf("-l emitter also showed %s, which belongs to another phase", other)
		}
	}
}

// Asking for nothing writes nothing: the traces are opt-in, and a compile that was not
// asked to explain itself stays quiet.
func TestNoLoggerWritesNothing(t *testing.T) {
	if written := compileWithLoggers(t, "ident a = 1;\n", nil); written != "" {
		t.Errorf("compiling with no logger wrote:\n%s", written)
	}
}

// A caller with nowhere to write is not an error — it is how `aurora test` compiles.
func TestCompileWithoutATraceWriter(t *testing.T) {
	entry := filepath.Join(t.TempDir(), "main.ar")
	if err := os.WriteFile(entry, []byte("ident a = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Compile(entry, 0, []string{"lexer", "parser", "emitter"}, nil, nil); err != nil {
		t.Errorf("compiling with every logger and no writer: %v", err)
	}
}
