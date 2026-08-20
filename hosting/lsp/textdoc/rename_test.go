package textdoc

import (
	"errors"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
)

// applied is the document with the rename carried out, which is what the editor ends up
// showing. Reading the result instead of the ranges is the only way to see the two mistakes
// that matter: an occurrence left behind, and one taken that meant something else.
func applied(t *testing.T, source string, ranges []lsp.Range, newName string) string {
	t.Helper()

	mapper := lsp.NewMapper(source)
	out := source
	// Backwards, so an edit does not move the ones not made yet.
	for i := len(ranges) - 1; i >= 0; i-- {
		start := mapper.Offset(ranges[i].Start)
		end := mapper.Offset(ranges[i].End)
		if start > end || end > len(out) {
			t.Fatalf("range %+v is not inside the document", ranges[i])
		}
		out = out[:start] + newName + out[end:]
	}
	return out
}

// A name bound inside a scope, renamed everywhere it is written and nowhere else.
func TestRenameInsideAScope(t *testing.T) {
	const source = "ident area = defer {\n" +
		"  ident side = feed(0);\n" +
		"  side * side;\n" +
		"};\n" +
		"printd area(4);\n"

	got, err := session().RenameFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 3}, "edge")
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if len(got.Ranges) != 3 {
		t.Fatalf("found %d places, want the declaration and its two readings: %+v", len(got.Ranges), got.Ranges)
	}

	want := "ident area = defer {\n" +
		"  ident edge = feed(0);\n" +
		"  edge * edge;\n" +
		"};\n" +
		"printd area(4);\n"
	if out := applied(t, source, got.Ranges, "edge"); out != want {
		t.Errorf("renaming gives:\n%s\nwant:\n%s", out, want)
	}
}

// The same name in two scopes is two names. Renaming one leaves the other alone, which is
// the mistake a search-and-replace makes and a scope walk does not.
func TestRenameTakesOnlyItsOwnScope(t *testing.T) {
	const source = "ident outer = defer {\n" +
		"  ident n = 1;\n" +
		"  n;\n" +
		"};\n" +
		"ident other = defer {\n" +
		"  ident n = 2;\n" +
		"  n + n;\n" +
		"};\n"

	got, err := session().RenameFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 5, Character: 8}, "count")
	if err != nil {
		t.Fatalf("refused: %v", err)
	}

	out := applied(t, source, got.Ranges, "count")
	if !strings.Contains(out, "ident count = 2;\n  count + count;") {
		t.Errorf("the second scope was not renamed:\n%s", out)
	}
	if !strings.Contains(out, "ident n = 1;\n  n;") {
		t.Errorf("the first scope was renamed too:\n%s", out)
	}
}

// An alias belongs to the file that wrote it and to no other, so it is the one name that can
// always be renamed — the use line and every reach through it.
func TestRenameAnAlias(t *testing.T) {
	session := withModuleFiles(map[string]string{"src/geometry.ar": "ident area = defer { feed(0); };\n"})
	const source = "use geometry as g;\nprintd g.area(2);\nprintd g.area(3);\n"

	got, err := session.RenameFor(Document{Filename: "src/main.ar", Source: source}, lsp.Position{Line: 1, Character: 7}, "geo")
	if err != nil {
		t.Fatalf("refused: %v", err)
	}

	want := "use geometry as geo;\nprintd geo.area(2);\nprintd geo.area(3);\n"
	if out := applied(t, source, got.Ranges, "geo"); out != want {
		t.Errorf("renaming gives:\n%s\nwant:\n%s", out, want)
	}
}

// The path of a use line is a module's name, not this file's, so nothing in it is renamed by
// asking about the alias — and asking about the path itself is refused.
func TestRenameDoesNotTouchTheModulePath(t *testing.T) {
	session := withModuleFiles(map[string]string{"src/geometry.ar": "ident area = defer { feed(0); };\n"})
	doc := Document{Filename: "src/main.ar", Source: "use geometry as g;\nprintd g.area(2);\n"}

	if _, err := session.RenameFor(doc, lsp.Position{Line: 0, Character: 6}, "shapes"); !errors.Is(err, ErrNotRenameable) {
		t.Errorf("renaming the module path answered %v, want a refusal", err)
	}
	if _, err := session.RenameFor(doc, lsp.Position{Line: 1, Character: 11}, "surface"); !errors.Is(err, ErrNotRenameable) {
		t.Errorf("renaming a name of another module answered %v, want a refusal", err)
	}
}

// What a file offers, another file may already be reaching for. Renaming it here would be
// half of the change, and the half that is missing is in a file nobody has looked at.
func TestRenameRefusesWhatLeavesTheFile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		at     lsp.Position
		says   string
	}{
		{
			name:   "a value bound at the top",
			source: "ident total = 1;\nprintd total;\n",
			at:     lsp.Position{Line: 1, Character: 8},
			says:   "top of this file",
		},
		{
			name:   "a shape declared at the top",
			source: "shape Point { x, y };\nprintd Point{1, 2};\n",
			at:     lsp.Position{Line: 1, Character: 8},
			says:   "top of this file",
		},
		{
			name:   "a field, which belongs to a shape",
			source: "shape Point { x, y };\nident p = Point{1, 2};\nprintd p.y;\n",
			at:     lsp.Position{Line: 2, Character: 9},
			says:   "not declared in this file",
		},
		{
			name:   "something that is not a name",
			source: "printd 42;\n",
			at:     lsp.Position{Line: 0, Character: 8},
			says:   "no name here",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := session().RenameFor(Document{Filename: "main.ar", Source: tc.source}, tc.at, "other")
			if !errors.Is(err, ErrNotRenameable) {
				t.Fatalf("answered %v, want a refusal", err)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("said %q, want it to say %q", err, tc.says)
			}
		})
	}
}

// A new name the language would refuse is refused here, before every occurrence is rewritten
// and the file stops compiling.
func TestRenameRefusesANameTheLanguageWouldNot(t *testing.T) {
	const source = "ident area = defer {\n  ident side = feed(0);\n  side;\n};\n"

	for _, newName := range []string{"defer", "2", "two words", "a.b", ""} {
		if _, err := session().RenameFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 3}, newName); !errors.Is(err, ErrNotRenameable) {
			t.Errorf("renaming to %q answered %v, want a refusal", newName, err)
		}
	}
}

// The editor asks before it opens its box, so the refusal arrives before somebody types.
func TestPrepareRename(t *testing.T) {
	const source = "ident area = defer {\n  ident side = feed(0);\n  side;\n};\n"
	doc := Document{Filename: "main.ar", Source: source}

	at, err := session().PrepareRename(doc, lsp.Position{Line: 2, Character: 3})
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if at != lsp.LineRange(2, 2, 6) {
		t.Errorf("points at %+v, want the name under the cursor", at)
	}

	if _, err := session().PrepareRename(doc, lsp.Position{Line: 0, Character: 8}); !errors.Is(err, ErrNotRenameable) {
		t.Errorf("preparing a name that leaves the file answered %v, want a refusal", err)
	}
}

// A scope ends at its brace. A name declared inside one shadows the file's while it is open
// and gives it back afterwards, so renaming the inner one leaves the readings under the
// block alone.
func TestRenameStopsAtTheEndOfTheScope(t *testing.T) {
	const source = "ident n = 1;\n" +
		"ident f = defer {\n" +
		"  ident n = 2;\n" +
		"  n * n;\n" +
		"};\n" +
		"printd n;\n"

	got, err := session().RenameFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 8}, "inner")
	if err != nil {
		t.Fatalf("refused: %v", err)
	}

	want := "ident n = 1;\n" +
		"ident f = defer {\n" +
		"  ident inner = 2;\n" +
		"  inner * inner;\n" +
		"};\n" +
		"printd n;\n"
	if out := applied(t, source, got.Ranges, "inner"); out != want {
		t.Errorf("renaming gives:\n%s\nwant:\n%s", out, want)
	}
}

// A deferred scope runs when it is called, so its body may name something written under it.
// That reading is part of the rename, and leaving it behind would stop the file compiling.
func TestRenameTakesAReadingWrittenAboveTheBinding(t *testing.T) {
	const source = "ident outer = defer {\n" +
		"  ident show = defer { later; };\n" +
		"  ident later = 5;\n" +
		"  show();\n" +
		"};\n"

	got, err := session().RenameFor(Document{Filename: "main.ar", Source: source}, lsp.Position{Line: 2, Character: 8}, "afterwards")
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	out := applied(t, source, got.Ranges, "afterwards")
	if strings.Contains(out, "later") {
		t.Errorf("a reading was left behind:\n%s", out)
	}
}
