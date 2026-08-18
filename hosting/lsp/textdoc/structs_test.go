package textdoc

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
)

const structSource = `struct Point { x, y };
struct Named { label, value };

ident p = Point{10, 20};
ident n = feed(0) as Named;
`

// Completing after a dot is the reason the declaration is worth having at all for whoever is
// typing: the fields are what to offer, and nothing else.
func TestCompletionAfterADotOffersTheFields(t *testing.T) {
	cases := []struct {
		name   string
		source string
		pos    lsp.Position
		want   []string
	}{
		{
			name:   "a name built from a construction",
			source: structSource + "p.",
			pos:    lsp.Position{Line: 5, Character: 2},
			want:   []string{"x", "y"},
		},
		{
			name:   "a name given a shape with as",
			source: structSource + "n.",
			pos:    lsp.Position{Line: 5, Character: 2},
			want:   []string{"label", "value"},
		},
		{
			name:   "halfway through typing the field",
			source: structSource + "p.x",
			pos:    lsp.Position{Line: 5, Character: 3},
			want:   []string{"x", "y"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := CompletionItemsFor(Document{Filename: "main.ar", Source: tc.source}, tc.pos, false)

			got := make([]string, 0, len(items))
			for _, item := range items {
				got = append(got, item.Label)
				if item.Kind != Field {
					t.Errorf("%s came back as kind %d, want a field", item.Label, item.Kind)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("offered %v, want %v", got, tc.want)
			}
		})
	}
}

// A document being edited hardly ever parses — `p.` on its own does not — and that is
// exactly when completion is asked for. The fields come from the tokens for that reason.
func TestCompletionAfterADotWorksWhileBroken(t *testing.T) {
	source := structSource + "printd p."
	if diagnostics := ValidateCode(Document{Filename: "main.ar", Source: source}); len(diagnostics) == 0 {
		t.Fatal("this source is supposed not to parse; the test is about that")
	}

	items := CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 5, Character: 9}, false)
	if len(items) != 2 || items[0].Label != "x" {
		t.Errorf("offered %v, want the fields of Point", items)
	}
}

// Anywhere else the ordinary list is what comes back.
func TestCompletionAwayFromADotIsUnchanged(t *testing.T) {
	items := CompletionItemsFor(Document{Filename: "main.ar", Source: structSource}, lsp.Position{Line: 5, Character: 0}, false)

	var hasKeyword bool
	for _, item := range items {
		if item.Kind == Keyword {
			hasKeyword = true
		}
		if item.Kind == Field {
			t.Errorf("%s was offered as a field away from a dot", item.Label)
		}
	}
	if !hasKeyword {
		t.Error("expected the keywords among the completions")
	}
}

// Hover is the other half: what the fields are, and which tape one of them reads.
func TestHoverDescribesStructsAndFields(t *testing.T) {
	cases := []struct {
		name string
		pos  lsp.Position
		want string
	}{
		{name: "the struct in its declaration", pos: lsp.Position{Line: 0, Character: 8}, want: "fields: x, y"},
		{name: "the struct named by as", pos: lsp.Position{Line: 4, Character: 21}, want: "fields: label, value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HoverInfo(Document{Filename: "main.ar", Source: structSource}, tc.pos); !strings.Contains(got, tc.want) {
				t.Errorf("hover = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestHoverOnAFieldSaysWhichTapeItReads(t *testing.T) {
	source := structSource + "printd p.y;"
	got := HoverInfo(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 5, Character: 9})

	for _, want := range []string{"field y", "struct Point", "tape 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("hover = %q, want it to contain %q", got, want)
		}
	}
}

// struct and as colour as keywords, a field as a property, and the struct's own name as a
// struct — the three places a name is not just a value.
func TestSemanticTokensForStructs(t *testing.T) {
	tokens := decode(SemanticTokensFor("struct Point { x, y };\nident p = Point{1, 2};\nprintd p.x;\n"))

	// struct, then the name it declares, then the two fields it names.
	want := []uint{SemanticKeyword, SemanticStruct, SemanticProperty, SemanticProperty}
	for i := range want {
		if got := tokens[i].tokenType; got != want[i] {
			t.Errorf("token %d is type %d, want %d", i, got, want[i])
		}
	}
}
