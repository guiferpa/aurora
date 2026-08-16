package textdoc

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lsp"
)

// find returns the item offered for a label, or nil.
func find(items []CompletionItem, label string) *CompletionItem {
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	return nil
}

// The forms worth expanding are the ones with a shape to get wrong: where the semicolon
// goes, that a branch ends in a fallback with no test, that a defer is a block.
func TestKeywordSnippets(t *testing.T) {
	cases := []struct {
		keyword string
		want    string
	}{
		{keyword: "defer", want: "defer {\n\t$0\n}"},
		{keyword: "ident", want: "ident ${1:name} = ${0:value};"},
		{keyword: "if", want: "if ${1:condition} {\n\t$0\n}"},
		{keyword: "branch", want: "branch {\n\t${1:test}: ${2:value},\n\t${0:fallback};\n}"},
		{keyword: "struct", want: "struct ${1:Name} { ${0:field} };"},
		{keyword: "assert", want: `assert(${1:condition}, "${0:message}");`},
		{keyword: "feed", want: "feed(${0:0})"},
		{keyword: "printb", want: "printb ${0:value};"},
		{keyword: "printc", want: "printc ${0:value};"},
		{keyword: "printd", want: "printd ${0:value};"},
		{keyword: "head", want: "head ${1:tape} ${0:1}"},
		{keyword: "pull", want: "pull ${1:tape} ${0:value}"},
	}

	items := CompletionItemsFor(Document{Filename: "main.ar", Source: ""}, lsp.Position{}, true)
	for _, tc := range cases {
		t.Run(tc.keyword, func(t *testing.T) {
			item := find(items, tc.keyword)
			if item == nil {
				t.Fatalf("%s is not offered at all", tc.keyword)
			}
			if item.InsertText != tc.want {
				t.Errorf("inserts %q, want %q", item.InsertText, tc.want)
			}
			if item.InsertTextFormat != SnippetFormat {
				t.Errorf("format is %d, want %d", item.InsertTextFormat, SnippetFormat)
			}
		})
	}
}

// A client that does not expand snippets gets the keyword and nothing else: the
// placeholders would land in the buffer as the literal text they are.
func TestNoSnippetsForAClientThatCannotExpandThem(t *testing.T) {
	items := CompletionItemsFor(Document{Filename: "main.ar", Source: "struct Point { x, y };"}, lsp.Position{}, false)

	for _, item := range items {
		if item.InsertText != "" || item.InsertTextFormat != 0 {
			t.Errorf("%s came back with %q as a snippet", item.Label, item.InsertText)
		}
	}
	if find(items, "defer") == nil {
		t.Error("the keywords are still offered")
	}
}

// A declared struct is offered as a way of building one, and the directive already said
// what the fields are — so they become the places to fill in.
func TestStructCompletion(t *testing.T) {
	const source = "struct Point { x, y };\nstruct Named { label, value, tag };\n"

	items := CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2}, true)

	point := find(items, "Point")
	if point == nil {
		t.Fatal("Point is not offered")
	}
	if point.InsertText != "Point{${1:x}, ${2:y}}" {
		t.Errorf("inserts %q", point.InsertText)
	}
	if point.Kind != Struct {
		t.Errorf("kind is %d, want a struct", point.Kind)
	}
	if !strings.Contains(point.Detail, "x, y") {
		t.Errorf("detail is %q, want it to name the fields", point.Detail)
	}

	named := find(items, "Named")
	if named == nil {
		t.Fatal("Named is not offered")
	}
	if named.InsertText != "Named{${1:label}, ${2:value}, ${3:tag}}" {
		t.Errorf("inserts %q", named.InsertText)
	}
}

// Fields are names and nothing else, so they are inserted as they are even where snippets
// are expanded.
func TestFieldsAreNotSnippets(t *testing.T) {
	source := "struct Point { x, y };\nident p = Point{1, 2};\np."
	items := CompletionItemsFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 2}, true)

	if len(items) != 2 {
		t.Fatalf("offered %d items, want the two fields", len(items))
	}
	for _, item := range items {
		if item.InsertText != "" {
			t.Errorf("field %s came back as a snippet %q", item.Label, item.InsertText)
		}
	}
}
