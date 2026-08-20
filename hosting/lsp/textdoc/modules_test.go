package textdoc

import (
	"fmt"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// What one file says about the modules it brings in, which is all the editor knows: that a
// name is a module rather than a value, and which module it is. What is inside one lives in
// another file, and nothing here has read it.

const withModules = "use a/b as x;\nuse other as y;\nident n = 1;\nprintd x.area(n);\n"

// A use line is read out of the tokens, so it is read while the document is broken too — and
// a half-written one says nothing rather than something wrong.
func TestScanUses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   map[string]string
	}{
		{
			name:   "one module",
			source: "use a/b/c as x;",
			want:   map[string]string{"x": "a/b/c"},
		},
		{
			name:   "several, and a path of one segment",
			source: "use a/b as x;\nuse other as y;",
			want:   map[string]string{"x": "a/b", "y": "other"},
		},
		{
			name:   "still being typed",
			source: "use a/b",
			want:   map[string]string{},
		},
		{
			name:   "typed as far as the alias",
			source: "use a/b as ",
			want:   map[string]string{},
		},
		{
			name:   "a document with none",
			source: "ident a = 1;",
			want:   map[string]string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := aliasesOf(parser.ScanUses(session().Analyze(Document{Filename: "main.ar", Source: tc.source}).Tokens))
			if len(got) != len(tc.want) {
				t.Fatalf("read %v, want %v", got, tc.want)
			}
			for alias, specifier := range tc.want {
				if got[alias] != specifier {
					t.Errorf("alias %s means %q, want %q", alias, got[alias], specifier)
				}
			}
		})
	}
}

// An alias is a module, and saying "identifier" about it was the wrong answer it used to give.
func TestHoverOnAModule(t *testing.T) {
	for _, tc := range []struct {
		name string
		pos  lsp.Position
		want string
	}{
		{name: "the alias", pos: lsp.Position{Line: 3, Character: 7}, want: "module a/b"},
		{name: "a name reached through it", pos: lsp.Position{Line: 3, Character: 10}, want: "of module a/b"},
		{name: "an ordinary name is untouched", pos: lsp.Position{Line: 3, Character: 14}, want: "identifier: n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := session().HoverInfo(Document{Filename: "main.ar", Source: withModules}, tc.pos)
			if !strings.Contains(got, tc.want) {
				t.Errorf("hover = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

// After a dot on a module, the editor has nothing to offer — and offering the keywords, which
// is what it did, is offering the one thing that certainly cannot go there.
func TestCompletionAfterAModuleDot(t *testing.T) {
	source := "use a/b as x;\nprintd x.\n"
	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 1, Character: 9}, false)

	if len(items) != 0 {
		t.Errorf("offered %d items after a module dot, want none: %+v", len(items), items)
	}
}

// A struct still answers with its fields after a dot, which is the other reader of the same
// question.
func TestCompletionAfterAStructDotStillWorks(t *testing.T) {
	source := "struct Point { x, y };\nident p = Point{1, 2};\nprintd p.\n"
	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 9}, false)

	if len(items) != 2 {
		t.Fatalf("offered %d items, want the two fields: %+v", len(items), items)
	}
	if items[0].Label != "x" || items[1].Label != "y" {
		t.Errorf("offered %q and %q, want x and y", items[0].Label, items[1].Label)
	}
}

// And the aliases themselves are offered, as modules rather than as values.
func TestCompletionOffersTheModules(t *testing.T) {
	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: withModules}, lsp.Position{Line: 2, Character: 0}, false)

	found := make(map[string]CompletionItem)
	for _, item := range items {
		found[item.Label] = item
	}
	for alias, specifier := range map[string]string{"x": "a/b", "y": "other"} {
		item, ok := found[alias]
		if !ok {
			t.Errorf("the alias %s was not offered", alias)
			continue
		}
		if item.Kind != Module {
			t.Errorf("the alias %s is offered as kind %d, want a module", alias, item.Kind)
		}
		if !strings.Contains(item.Detail, specifier) {
			t.Errorf("the alias %s is described as %q, want it to name %s", alias, item.Detail, specifier)
		}
	}
}

// From here on the editor reads the other files too, through the port it is handed. These
// hand it a project in memory: what is on a disk is the server's business, and this package
// answers with values whatever the world it is put in.
func withModuleFiles(files map[string]string) *Session {
	lx := lexer.New()
	ps := parser.New()

	return NewSession(NewSessionOptions{
		Lexer:  lx,
		Parser: ps,
		Resolve: func(doc Document, uses []ast.UseDeclaration) ([]module.Module, error) {
			return resolver.New(resolver.Options{
				SourceRoot: "src",
				Read: func(path string) ([]byte, error) {
					source, ok := files[path]
					if !ok {
						return nil, fmt.Errorf("no such file")
					}
					return []byte(source), nil
				},
				Parse: func(filename string, id module.ID, source []byte, imports map[string][]ast.Promise) (ast.AST, error) {
					tokens, err := lx.GetFilledTokens(source)
					if err != nil {
						return ast.AST{}, err
					}
					return ps.Parse(parser.ParseInput{Filename: filename, Tokens: tokens, Module: string(id), Imports: imports})
				},
				Header: func(source []byte) ([]ast.UseDeclaration, error) {
					tokens, err := lx.GetFilledTokens(source)
					if err != nil {
						return nil, err
					}
					return parser.ScanUses(tokens), nil
				},
			}).DependenciesOf(doc.Filename, uses)
		},
	})
}

const geometry = "ident area = defer { feed(0) * feed(1); };\nident base = 10;"

// What the imports say is underlined where it was written, and the message is the compiler's
// own — the editor adds nothing to it.
func TestModuleDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
		line   int
	}{
		{
			name:   "a module that is not there",
			source: "use a/b as x;\nprintd 1;",
			want:   "module a/b is not there",
			line:   0,
		},
		{
			name:   "a name the module does not have",
			source: "use geometry as g;\nprintd g.volume(1, 2);",
			want:   "module geometry has no volume",
			line:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := withModuleFiles(map[string]string{"src/geometry.ar": geometry})
			diagnostics := session.ValidateCode(Document{Filename: "src/main.ar", Source: tc.source})

			if len(diagnostics) != 1 {
				t.Fatalf("reported %d diagnostics, want 1: %+v", len(diagnostics), diagnostics)
			}
			if !strings.Contains(diagnostics[0].Message, tc.want) {
				t.Errorf("said %q, want it to say %q", diagnostics[0].Message, tc.want)
			}
			if diagnostics[0].Range.Start.Line != tc.line {
				t.Errorf("underlined line %d, want %d", diagnostics[0].Range.Start.Line, tc.line)
			}
		})
	}
}

// A program whose imports are all there says nothing, which is the case that matters most.
func TestNoDiagnosticWhenTheModulesAreThere(t *testing.T) {
	session := withModuleFiles(map[string]string{"src/geometry.ar": geometry})
	diagnostics := session.ValidateCode(Document{
		Filename: "src/main.ar",
		Source:   "use geometry as g;\nprintd g.area(2, 3);",
	})

	if len(diagnostics) != 0 {
		t.Errorf("reported %+v, want nothing", diagnostics)
	}
}

// After the dot, what the module declared — and only that.
func TestCompletionOffersWhatTheModuleDeclared(t *testing.T) {
	session := withModuleFiles(map[string]string{"src/geometry.ar": geometry})
	items := session.CompletionItemsFor(Document{
		Filename: "src/main.ar",
		Source:   "use geometry as g;\nprintd g.\n",
	}, lsp.Position{Line: 1, Character: 9}, false)

	if len(items) != 2 {
		t.Fatalf("offered %d items, want the two the module declared: %+v", len(items), items)
	}
	for i, want := range []string{"area", "base"} {
		if items[i].Label != want {
			t.Errorf("offered %q, want %q", items[i].Label, want)
		}
		if !strings.Contains(items[i].Detail, "geometry") {
			t.Errorf("described %q as %q, want it to name the module", items[i].Label, items[i].Detail)
		}
	}
	if !strings.Contains(items[0].Detail, "deferred scope") {
		t.Errorf("area is described as %q, want it to say what it is", items[0].Detail)
	}
}

// A module that could not be found offers nothing rather than everything: the diagnostic
// already says it is missing.
func TestCompletionAfterAMissingModule(t *testing.T) {
	session := withModuleFiles(map[string]string{})
	items := session.CompletionItemsFor(Document{
		Filename: "src/main.ar",
		Source:   "use gone as g;\nprintd g.\n",
	}, lsp.Position{Line: 1, Character: 9}, false)

	if len(items) != 0 {
		t.Errorf("offered %d items for a module that is not there", len(items))
	}
}

// Hover reads the other file too, so a name reached through a module says what it is.
func TestHoverReadsTheModule(t *testing.T) {
	session := withModuleFiles(map[string]string{"src/geometry.ar": geometry})
	got := session.HoverInfo(Document{
		Filename: "src/main.ar",
		Source:   "use geometry as g;\nprintd g.area(2, 3);",
	}, lsp.Position{Line: 1, Character: 10})

	for _, want := range []string{"of module geometry", "deferred scope"} {
		if !strings.Contains(got, want) {
			t.Errorf("hover = %q, want it to contain %q", got, want)
		}
	}
}

// Without the port a document is read on its own, which is what a page with one editor in it
// is: nothing is missing, because nothing could have been imported.
func TestWithoutThePortNothingIsResolved(t *testing.T) {
	diagnostics := session().ValidateCode(Document{
		Filename: "main.ar",
		Source:   "use a/b as x;\nprintd 1;",
	})

	if len(diagnostics) != 0 {
		t.Errorf("reported %+v, want nothing said about a module nobody looked for", diagnostics)
	}
}
