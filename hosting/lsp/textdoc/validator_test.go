package textdoc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func TestValidateCodeAcceptsValidSource(t *testing.T) {
	diagnostics := session().ValidateCode(Document{Filename: "main.ar", Source: "ident a = 1;\nprintb a + 1;\n"})
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diagnostics)
	}
}

func TestValidateCodeReportsPositions(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		line      int
		character int
		message   string
	}{
		{
			name:      "parser error points at the offending token",
			source:    "ident a = 1;\nident = 2;\n",
			line:      1,
			character: 6,
			message:   "unexpected token",
		},
		{
			name:      "lexer error points at the character",
			source:    "ident a = 1;\nident b = @;\n",
			line:      1,
			character: 10,
			message:   "unexpected character",
		},
		{
			// The error lands on the EOF token, past the end of the text. Rather than a
			// zero-width marker no client would draw, it underlines the last meaningful
			// character — exactly where the semicolon belongs.
			name:      "missing semicolon",
			source:    "ident a = 1\n",
			line:      0,
			character: 10,
			message:   "unexpected token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := session().ValidateCode(Document{Filename: "main.ar", Source: tc.source})
			if len(diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d: %v", len(diagnostics), diagnostics)
			}
			got := diagnostics[0]
			if got.Severity != SeverityError {
				t.Errorf("severity = %d, want %d", got.Severity, SeverityError)
			}
			if !strings.Contains(got.Message, tc.message) {
				t.Errorf("message = %q, want it to contain %q", got.Message, tc.message)
			}
			if got.Range.Start.Line != tc.line || got.Range.Start.Character != tc.character {
				t.Errorf("start = %d:%d, want %d:%d",
					got.Range.Start.Line, got.Range.Start.Character, tc.line, tc.character)
			}
			if got.Range.End.Character <= got.Range.Start.Character && got.Range.End.Line == got.Range.Start.Line {
				t.Errorf("range is empty: %+v", got.Range)
			}
		})
	}
}

// The parser only accepts "assert" in *.test.ar, so the filename behind the URI has to
// reach it — otherwise a valid test file would be reported as broken.
func TestValidateCodeHonoursTestFileRule(t *testing.T) {
	const source = "assert(1 equals 1, \"ok\");\n"

	if diagnostics := session().ValidateCode(Document{Filename: "math.test.ar", Source: source}); len(diagnostics) != 0 {
		t.Errorf("assert should be accepted in a .test.ar file, got %v", diagnostics)
	}
	diagnostics := session().ValidateCode(Document{Filename: "math.ar", Source: source})
	if len(diagnostics) != 1 {
		t.Fatalf("assert should be rejected outside .test.ar, got %v", diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, ".test.ar") {
		t.Errorf("message = %q, want it to mention .test.ar", diagnostics[0].Message)
	}
}

func TestHoverInfo(t *testing.T) {
	const source = "ident a = 1;\nident sum = defer { feed(0); };\nprintb a;\n"

	cases := []struct {
		name string
		pos  lsp.Position
		want string
	}{
		{name: "keyword uses the tag description", pos: lsp.Position{Line: 0, Character: 2}, want: "immutable identifier"},
		{name: "number", pos: lsp.Position{Line: 0, Character: 10}, want: "number: 1"},
		{name: "identifier resolves to its declaration", pos: lsp.Position{Line: 2, Character: 7}, want: "identifier: a"},
		{name: "deferred scope is described", pos: lsp.Position{Line: 1, Character: 7}, want: "deferred scope"},
		{name: "whitespace has nothing to say", pos: lsp.Position{Line: 0, Character: 5}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := session().HoverInfo(Document{Filename: "main.ar", Source: source}, tc.pos)
			if tc.want == "" {
				if got != "" {
					t.Errorf("hover = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("hover = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestCompletionItemsIncludeKeywordsAndDocumentIdents(t *testing.T) {
	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: "ident total = 1;\n"}, lsp.Position{Line: 1, Character: 0}, false)

	var hasKeyword, hasIdent bool
	for _, item := range items {
		if item.Label == "ident" && item.Kind == Keyword {
			hasKeyword = true
		}
		if item.Label == "total" && item.Kind == Variable {
			hasIdent = true
		}
	}
	if !hasKeyword {
		t.Error("expected the ident keyword among the completions")
	}
	if !hasIdent {
		t.Error("expected the declared identifier among the completions")
	}
}

func TestCompletionItemsSurviveBrokenSource(t *testing.T) {
	if items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: "ident @@@"}, lsp.Position{Line: 0, Character: 9}, false); len(items) == 0 {
		t.Error("keywords should still be offered while the document does not parse")
	}
}

func TestPathFromURI(t *testing.T) {
	cases := []struct {
		name string
		uri  lsp.URI
		want string
	}{
		{name: "plain path", uri: "file:///home/dev/main.ar", want: filepath.FromSlash("/home/dev/main.ar")},
		{name: "percent-encoded space", uri: "file:///home/dev/my%20project/main.test.ar", want: filepath.FromSlash("/home/dev/my project/main.test.ar")},
		{name: "not a file uri", uri: "untitled:Untitled-1", want: "untitled:Untitled-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PathFromURI(tc.uri); got != tc.want {
				t.Errorf("PathFromURI(%q) = %q, want %q", tc.uri, got, tc.want)
			}
		})
	}
}

// A document that parses can still be worth a word, and the editor is where it is cheapest to
// hear. Until this the analysis stopped at the parser, so everything the compiler had to say
// short of refusing was said only to whoever ran "aurora build".
func TestADocumentThatParsesCanStillBeWarnedAbout(t *testing.T) {
	session := NewSession(NewSessionOptions{
		Lexer:  lexer.New(),
		Parser: parser.New(),
		Emit:   emitter.New(emitter.NewEmitterOptions{}).EmitProgram,
	})

	source := "ident fn = defer {\n  printd feed(2);\n};\n\nfn();\n"
	diagnostics := session.ValidateCode(Document{Filename: "main.ar", Source: source})

	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diagnostics)
	}
	if got := diagnostics[0].Severity; got != SeverityWarning {
		t.Errorf("severity is %d, want a warning: a short call is legal, and what was not applied answers zeros", got)
	}
	if !strings.Contains(diagnostics[0].Message, "3 positions") {
		t.Errorf("message is %q, want it to say the scope reads three positions", diagnostics[0].Message)
	}
	// Line 5 in the source, which the editor counts from zero.
	if got := diagnostics[0].Range.Start.Line; got != 4 {
		t.Errorf("it underlines line %d, want the line the call is on", got)
	}
}

// Without the port a document is checked as far as it parses, which is what the playground
// does and what this was before.
func TestWithoutTheEmitterOnlyTheParseIsChecked(t *testing.T) {
	session := NewSession(NewSessionOptions{Lexer: lexer.New(), Parser: parser.New()})

	source := "ident fn = defer {\n  printd feed(2);\n};\n\nfn();\n"
	if diagnostics := session.ValidateCode(Document{Filename: "main.ar", Source: source}); len(diagnostics) != 0 {
		t.Errorf("expected nothing to say, got %v", diagnostics)
	}
}

// An editor validates on every keystroke, so what one pass costs is worth knowing rather than
// assuming. The two benchmarks are the same document through the same session, with and
// without the emitter, and the difference between them is what asking the compiler costs.
func benchmarkValidate(b *testing.B, emit Emit) {
	var source strings.Builder
	source.WriteString("ident base = 10;\n")
	for i := 0; i < 200; i++ {
		source.WriteString("ident scope = defer { feed(0) + feed(1) * 2; };\n")
		source.WriteString("printd scope(1, 2);\n")
	}

	session := NewSession(NewSessionOptions{Lexer: lexer.New(), Parser: parser.New(), Emit: emit})
	doc := Document{Filename: "main.ar", Source: source.String()}

	b.ReportAllocs()
	for b.Loop() {
		session.ValidateCode(doc)
	}
}

func BenchmarkValidateCodeParseOnly(b *testing.B) {
	benchmarkValidate(b, nil)
}

func BenchmarkValidateCodeWithTheEmitter(b *testing.B) {
	benchmarkValidate(b, emitter.New(emitter.NewEmitterOptions{}).EmitProgram)
}

// The editor says what a chain would drop, too.
//
// It is a different question from the one the emitter answers — not what is wrong with the
// program, but what would be missing from the binary — and it was only ever asked by "aurora
// build", which is after the writing is done and often after the deciding is done too.
func TestTheEditorSaysWhatABackendWouldNotCarry(t *testing.T) {
	session := NewSession(NewSessionOptions{
		Lexer:   lexer.New(),
		Parser:  parser.New(),
		Emit:    emitter.New(emitter.NewEmitterOptions{}).EmitProgram,
		Carries: evm.Warnings,
	})

	source := "ident double = defer {\n  printd feed(0);\n  feed(0) * 2;\n};\n"
	diagnostics := session.ValidateCode(Document{Filename: "main.ar", Source: source})

	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diagnostics)
	}
	if got := diagnostics[0].Severity; got != SeverityWarning {
		t.Errorf("severity is %d, want a warning: the program is right, the binary is smaller", got)
	}
	if !strings.Contains(diagnostics[0].Message, "printd is ignored in compiled code") {
		t.Errorf("message is %q, want it to say the log does not reach the chain", diagnostics[0].Message)
	}
	// Line 2 in the source, which the editor counts from zero.
	if got := diagnostics[0].Range.Start.Line; got != 1 {
		t.Errorf("it underlines line %d, want the line the print is on", got)
	}
}

// Without the port the editor hears nothing about the binary. It is optional for the same
// reason the emitter's port is: a host that only wants a document checked as far as it parses
// should not have to carry a backend to get it.
func TestWithoutTheBackendPortNothingIsSaidAboutTheBinary(t *testing.T) {
	session := NewSession(NewSessionOptions{
		Lexer:  lexer.New(),
		Parser: parser.New(),
		Emit:   emitter.New(emitter.NewEmitterOptions{}).EmitProgram,
	})

	source := "ident double = defer {\n  printd feed(0);\n  feed(0) * 2;\n};\n"
	if diagnostics := session.ValidateCode(Document{Filename: "main.ar", Source: source}); len(diagnostics) != 0 {
		t.Errorf("said %v without being given a backend to ask", diagnostics)
	}
}
