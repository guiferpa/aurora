package textdoc

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
)

const shapeSource = `shape Point { x, y };
shape Named { label, value };

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
			source: shapeSource + "p.",
			pos:    lsp.Position{Line: 5, Character: 2},
			want:   []string{"x", "y"},
		},
		{
			name:   "a name given a shape with as",
			source: shapeSource + "n.",
			pos:    lsp.Position{Line: 5, Character: 2},
			want:   []string{"label", "value"},
		},
		{
			name:   "halfway through typing the field",
			source: shapeSource + "p.x",
			pos:    lsp.Position{Line: 5, Character: 3},
			want:   []string{"x", "y"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: tc.source}, tc.pos, false)

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
	source := shapeSource + "printd p."
	if diagnostics := session().ValidateCode(Document{Filename: "main.ar", Source: source}); len(diagnostics) == 0 {
		t.Fatal("this source is supposed not to parse; the test is about that")
	}

	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 5, Character: 9}, false)
	if len(items) != 2 || items[0].Label != "x" {
		t.Errorf("offered %v, want the fields of Point", items)
	}
}

// Anywhere else the ordinary list is what comes back.
func TestCompletionAwayFromADotIsUnchanged(t *testing.T) {
	items := session().CompletionItemsFor(Document{Filename: "main.ar", Source: shapeSource}, lsp.Position{Line: 5, Character: 0}, false)

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
func TestHoverDescribesShapesAndFields(t *testing.T) {
	cases := []struct {
		name string
		pos  lsp.Position
		want string
	}{
		{name: "the shape in its declaration", pos: lsp.Position{Line: 0, Character: 8}, want: "fields: x, y"},
		{name: "the shape named by as", pos: lsp.Position{Line: 4, Character: 21}, want: "fields: label, value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := session().HoverInfo(Document{Filename: "main.ar", Source: shapeSource}, tc.pos); !strings.Contains(got, tc.want) {
				t.Errorf("hover = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestHoverOnAFieldSaysWhichTapeItReads(t *testing.T) {
	source := shapeSource + "printd p.y;"
	got := session().HoverInfo(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 5, Character: 9})

	for _, want := range []string{"field y", "shape Point", "tape 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("hover = %q, want it to contain %q", got, want)
		}
	}
}

// shape and as colour as keywords, a field as a property, and the shape's own name as a
// shape — the three places a name is not just a value.
func TestSemanticTokensForShapes(t *testing.T) {
	tokens := decode(session().SemanticTokensFor("shape Point { x, y };\nident p = Point{1, 2};\nprintd p.x;\n"))

	// shape, then the name it declares, then the two fields it names.
	want := []uint{SemanticKeyword, SemanticShape, SemanticProperty, SemanticProperty}
	for i := range want {
		if got := tokens[i].tokenType; got != want[i] {
			t.Errorf("token %d is type %d, want %d", i, got, want[i])
		}
	}
}

// What a scope returns reaches the editor, and it reaches it from the compiler — even here,
// where the document does not parse.
//
// That is the point of handing the parser its table rather than letting it keep its own: it
// is filled as the file is read, so the three lines before the broken one are understood and
// only the broken one is lost. A document being edited is broken most of the time, and the
// moment somebody types a dot is the moment they want to be told what is there.
func TestTheEditorReadsWhatAScopeReturns(t *testing.T) {
	const source = "shape Result { failed, value };\n" +
		"ident divide = defer {\n" +
		"  if feed(1) equals 0 { Result{1, 0}; } else { Result{0, 1}; };\n" +
		"} returns Result;\n" +
		"ident r = divide(10, 2);\n" +
		"printd r."

	document := Document{Filename: "main.ar", Source: source}
	items := session().CompletionItemsFor(document, lsp.Position{Line: 5, Character: 9}, false)

	if len(items) != 2 {
		t.Fatalf("offered %d items after the dot, want the two fields: %+v", len(items), items)
	}
	for i, want := range []string{"failed", "value"} {
		if items[i].Label != want {
			t.Errorf("offered %q, want %q", items[i].Label, want)
		}
	}
}

// A scope that wrote no `returns` is understood too, because the editor reads what the
// compiler worked out and the compiler reads what the body ends with.
//
// The walk of the tokens never could: it matches `returns Result` and nothing else. This is
// the whole reason for asking the compiler first.
func TestTheEditorReadsAScopeThatDeclaredNothing(t *testing.T) {
	const source = "shape Result { failed, value };\n" +
		"ident divide = defer { Result{1, 0}; };\n" +
		"ident r = divide(10, 2);\n" +
		"printd r."

	document := Document{Filename: "main.ar", Source: source}
	items := session().CompletionItemsFor(document, lsp.Position{Line: 3, Character: 9}, false)

	if len(items) != 2 {
		t.Fatalf("offered %d items after the dot, want the two fields: %+v", len(items), items)
	}
	for i, want := range []string{"failed", "value"} {
		if items[i].Label != want {
			t.Errorf("offered %q, want %q", items[i].Label, want)
		}
	}
}

// And the walk of the tokens still answers where the compiler never arrived.
//
// The first line here is broken, so the parser reads nothing at all and its table is empty —
// which is the case the walk exists for. It knows less than the compiler does, and less is
// the whole of what it has to beat: completing a name has to work in a document that does
// not parse, which is most of them while somebody is typing.
func TestTheEditorFallsBackToTheTokens(t *testing.T) {
	const source = "ident broken = ;\n" +
		"shape Point { x, y };\n" +
		"ident p = feed(0) as Point;\n" +
		"printd p."

	document := Document{Filename: "main.ar", Source: source}

	// Said out loud, or the test passes for the wrong reason: both sources would offer x and
	// y here, and this is the one that has to.
	if left := session().Analyze(document).Declarations; len(left.Shapes) != 0 || len(left.Reads) != 0 {
		t.Fatalf("the parser got somewhere after all: %v %v", left.Shapes, left.Reads)
	}

	items := session().CompletionItemsFor(document, lsp.Position{Line: 3, Character: 9}, false)

	if len(items) != 2 {
		t.Fatalf("offered %d items after the dot, want the two fields: %+v", len(items), items)
	}
	for i, want := range []string{"x", "y"} {
		if items[i].Label != want {
			t.Errorf("offered %q, want %q", items[i].Label, want)
		}
	}
}

// The body of a scope is another scope's business: a construction inside it says what that
// scope returns, not what the name being bound is.
func TestABodyDoesNotShapeTheNameItIsBoundTo(t *testing.T) {
	const source = "shape Result { failed, value };\n" +
		"ident divide = defer { Result{1, 0}; };\n" +
		"printd divide."

	document := Document{Filename: "main.ar", Source: source}
	items := session().CompletionItemsFor(document, lsp.Position{Line: 2, Character: 14}, false)

	for _, item := range items {
		if item.Label == "failed" || item.Label == "value" {
			t.Errorf("offered %q: a deferred scope is not the shape its body builds", item.Label)
		}
	}
}
