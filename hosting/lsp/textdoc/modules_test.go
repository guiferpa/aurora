package textdoc

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
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
			got := scanUses(session().Analyze(Document{Filename: "main.ar", Source: tc.source}).Tokens)
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
