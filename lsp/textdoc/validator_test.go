package textdoc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lsp"
)

func TestValidateCodeAcceptsValidSource(t *testing.T) {
	diagnostics := ValidateCode("main.ar", "ident a = 1;\nprintb a + 1;\n")
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
			diagnostics := ValidateCode("main.ar", tc.source)
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

	if diagnostics := ValidateCode("math.test.ar", source); len(diagnostics) != 0 {
		t.Errorf("assert should be accepted in a .test.ar file, got %v", diagnostics)
	}
	diagnostics := ValidateCode("math.ar", source)
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
			got := HoverInfo("main.ar", source, tc.pos)
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
	items := CompletionItemsFor("main.ar", "ident total = 1;\n", lsp.Position{Line: 1, Character: 0})

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
	if items := CompletionItemsFor("main.ar", "ident @@@", lsp.Position{Line: 0, Character: 9}); len(items) == 0 {
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
