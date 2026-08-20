package textdoc

import (
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
)

// Where a name was declared, inside the file that uses it. The cursor goes on the name being
// asked about, and the answer is the range of the name where it was written.
func TestDefinitionInTheSameFile(t *testing.T) {
	const source = "ident width = 10;\n" +
		"shape Point { x, y };\n" +
		"ident p = Point{width, 2};\n" +
		"printd p.y;\n"

	for _, tc := range []struct {
		name string
		at   lsp.Position
		want lsp.Range
	}{
		{
			name: "a value, from where it is used",
			at:   lsp.Position{Line: 2, Character: 16},
			want: lsp.LineRange(0, 6, 11),
		},
		{
			name: "a shape, from a construction",
			at:   lsp.Position{Line: 2, Character: 11},
			want: lsp.LineRange(1, 6, 11),
		},
		{
			name: "a field, from where it is read",
			at:   lsp.Position{Line: 3, Character: 9},
			want: lsp.LineRange(1, 17, 18),
		},
		{
			name: "a name asked about where it is declared answers with itself",
			at:   lsp.Position{Line: 0, Character: 6},
			want: lsp.LineRange(0, 6, 11),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found, ok := session().DefinitionFor(Document{Filename: "main.ar", Source: source}, tc.at)
			if !ok {
				t.Fatal("nothing was found")
			}
			if found.Filename != "main.ar" {
				t.Errorf("points at %s, want the file itself", found.Filename)
			}
			if found.Range != tc.want {
				t.Errorf("points at %+v, want %+v", found.Range, tc.want)
			}
		})
	}
}

// Nothing is answered rather than something wrong. A jump the editor cannot make is a jump
// that does not happen; a jump to the wrong line is one somebody has to undo.
func TestDefinitionOfWhatIsNotDeclared(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		at     lsp.Position
	}{
		{
			name:   "a name nothing declares",
			source: "printd nope;\n",
			at:     lsp.Position{Line: 0, Character: 8},
		},
		{
			name:   "a number",
			source: "printd 42;\n",
			at:     lsp.Position{Line: 0, Character: 8},
		},
		{
			name:   "a keyword",
			source: "printd 42;\n",
			at:     lsp.Position{Line: 0, Character: 2},
		},
		{
			name:   "past the end of the document",
			source: "printd 42;\n",
			at:     lsp.Position{Line: 9, Character: 0},
		},
		{
			name:   "a field of a shape that has no such field",
			source: "shape Point { x, y };\nident p = Point{1, 2};\nprintd p.z;\n",
			at:     lsp.Position{Line: 2, Character: 9},
		},
		{
			name:   "a field of a value with no shape",
			source: "ident p = 1;\nprintd p.x;\n",
			at:     lsp.Position{Line: 1, Character: 9},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if found, ok := session().DefinitionFor(Document{Filename: "main.ar", Source: tc.source}, tc.at); ok {
				t.Errorf("answered %+v, want nothing", found)
			}
		})
	}
}

// A name bound twice is two names, and the one that answers is the one the language would
// find: the binding above the cursor, not the first in the file.
func TestDefinitionOfAShadowedName(t *testing.T) {
	const source = "ident n = 1;\n" +
		"printd n;\n" +
		"ident n = 2;\n" +
		"printd n;\n"

	first, ok := session().DefinitionFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 1, Character: 7})
	if !ok {
		t.Fatal("the first reading found nothing")
	}
	if first.Range != lsp.LineRange(0, 6, 7) {
		t.Errorf("the first reading points at %+v, want the binding above it", first.Range)
	}

	second, ok := session().DefinitionFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 3, Character: 7})
	if !ok {
		t.Fatal("the second reading found nothing")
	}
	if second.Range != lsp.LineRange(2, 6, 7) {
		t.Errorf("the second reading points at %+v, want the binding above it", second.Range)
	}
}

// A deferred scope runs when it is called, so its body can name something written under it.
// The binding below is a better answer than none.
func TestDefinitionOfANameBoundFurtherDown(t *testing.T) {
	const source = "ident show = defer { later; };\nident later = 5;\nprintd show();\n"

	found, ok := session().DefinitionFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 0, Character: 22})
	if !ok {
		t.Fatal("nothing was found")
	}
	if found.Range != lsp.LineRange(1, 6, 11) {
		t.Errorf("points at %+v, want the binding below it", found.Range)
	}
}
