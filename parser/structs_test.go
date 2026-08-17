package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/wire/token"
)

// Reading a field is resolved while parsing: the name becomes the index it was declared at,
// and the tree carries the index.
func TestFieldResolvesToItsIndex(t *testing.T) {
	cases := []struct {
		source string
		want   uint64
	}{
		{source: "struct Point { x, y };\nident p = Point{1, 2};\np.x;", want: 0},
		{source: "struct Point { x, y };\nident p = Point{1, 2};\np.y;", want: 1},
		{source: "struct Row { a, b, c };\nident r = Row{1, 2, 3};\nr.c;", want: 2},
		// The shape can come from the annotation instead of from a construction.
		{source: "struct Point { x, y };\nident p = feed(0) as Point;\np.y;", want: 1},
		// And it does not need a name at all.
		{source: "struct Point { x, y };\n(feed(0) as Point).y;", want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			nodes := parse(t, tc.source)
			field, ok := nodes[len(nodes)-1].(FieldExpression)
			if !ok {
				t.Fatalf("last node is %T, want a FieldExpression", nodes[len(nodes)-1])
			}
			if field.Index != tc.want {
				t.Errorf("index = %d, want %d", field.Index, tc.want)
			}
		})
	}
}

// Pointing at a mistake where it was written is what the directive is for, so these are the
// cases that justify having one at all. Every error carries a position, which is what puts
// the squiggle under the right word in an editor.
func TestStructDirectiveReportsMistakes(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "field that does not exist",
			source: "struct Point { x, y };\nident p = Point{1, 2};\np.z;",
			want:   "struct Point has no field named z",
		},
		{
			name:   "value with no shape",
			source: "struct Point { x, y };\nident p = 1;\np.x;",
			want:   "nothing says which struct this value is",
		},
		{
			name:   "too few values",
			source: "struct Point { x, y };\nPoint{1};",
			want:   "struct Point has 2 fields (x, y) but got 1",
		},
		{
			name:   "too many values",
			source: "struct Point { x, y };\nPoint{1, 2, 3};",
			want:   "but got 3",
		},
		{
			name:   "field declared twice",
			source: "struct Point { x, x };",
			want:   "already has a field named x",
		},
		{
			name:   "struct declared twice",
			source: "struct Point { x, y };\nstruct Point { a, b };",
			want:   "struct Point is already declared",
		},
		{
			name:   "no fields at all",
			source: "struct Empty { };",
			want:   "struct Empty has no fields",
		},
		{
			name:   "shape that was never declared",
			source: "struct Point { x, y };\nident p = 1 as Vector;",
			want:   "Vector is not a declared struct",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSource(t, tc.source, "main.ar")
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to say %q", err, tc.want)
			}
			positioned, ok := err.(*token.Error)
			if !ok {
				t.Fatalf("error is %T, want a positioned *token.Error", err)
			}
			if positioned.Line == 0 || positioned.Column == 0 {
				t.Errorf("error has no position: line %d, column %d", positioned.Line, positioned.Column)
			}
		})
	}
}

// A field is one tape wide, so it is never a struct itself and reading a field of a field
// has no shape to work from.
func TestFieldOfAFieldHasNoShape(t *testing.T) {
	_, err := parseSource(t, "struct Point { x, y };\nident p = Point{1, 2};\np.x.y;", "main.ar")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "nothing says which struct this value is") {
		t.Errorf("error = %q", err)
	}
}

// The directive says nothing about how a value is built, so a name bound to a struct can be
// used as an ordinary value everywhere else.
func TestStructNameIsOnlyADirective(t *testing.T) {
	for _, source := range []string{
		"struct Point { x, y };\nident p = Point{1, 2};\nprintd p + 1;",
		"struct Point { x, y };\nprintd Point{1, 2} equals Point{1, 2};",
		"struct Point { x, y };\nident p = Point{1, 2};\nident q = p;\nprintd q.y;",
	} {
		if _, err := parseSource(t, source, "main.ar"); err != nil {
			t.Errorf("parsing %q: %v", source, err)
		}
	}
}

// Braces build the value, as in Go. Parentheses could not: `Point(1, 2)` and `greet(1, 2)`
// are the same shape, so telling them apart needed the directive; `Point{1, 2}` does not.
func TestConstructionUsesBraces(t *testing.T) {
	nodes := parse(t, "struct Point { x, y };\nPoint{1, 2};")
	if _, ok := nodes[1].(StructLiteral); !ok {
		t.Errorf("Point{1, 2} parsed as %T, want a StructLiteral", nodes[1])
	}

	// A brace after a name is only a construction when a struct declared that name, or
	// `if flag { ... }` would stop parsing.
	if _, err := parseSource(t, "ident flag = true;\nif flag { 1; };", "main.ar"); err != nil {
		t.Errorf("an if on a plain name: %v", err)
	}

	// Parentheses still apply values to a scope.
	nodes = parse(t, "ident greet = defer { 1; };\ngreet();")
	if _, ok := nodes[1].(CalleeLiteral); !ok {
		t.Errorf("greet() parsed as %T, want a CalleeLiteral", nodes[1])
	}
}

// A struct name is a directive, not a value: there is nothing to load under it. The error
// says how to build one instead, which is what someone reaching for it wanted.
func TestStructNameIsNotAValue(t *testing.T) {
	_, err := parseSource(t, "struct Point { x, y };\nPoint;", "main.ar")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "Point is a struct") || !strings.Contains(err.Error(), "Point{...}") {
		t.Errorf("error = %q, want it to say Point is a struct and how to build one", err)
	}
}

// A field is one tape wide and text is a tape, so text goes in a field like anything else.
// What limits it is the width of a tape, and that is reported where the text was written.
func TestTextInAField(t *testing.T) {
	if _, err := parseSource(t, "struct Person { name, age };\nident p = Person{\"Gui\", 20};", "main.ar"); err != nil {
		t.Errorf("text that fits a tape: %v", err)
	}

	_, err := parseSource(t, "struct Person { name, age };\nident p = Person{\"Guilherme\", 20};", "main.ar")
	if err == nil {
		t.Fatal("expected a compile error: nine bytes do not fit an eight-byte tape")
	}
	if !strings.Contains(err.Error(), "text is 9 bytes but a tape holds 8") {
		t.Errorf("error = %q", err)
	}
}

// The directives belong to the file, not to one parse. The REPL compiles a line at a time
// with a fresh parser, so a struct declared on one line has to still be known on the next.
func TestDirectivesSurviveSeveralParses(t *testing.T) {
	directives := NewDirectives()

	parseWith := func(source string) error {
		tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
		if err != nil {
			return err
		}
		_, err = New(NewParserOptions{Filename: "main.ar", Tokens: tokens, Directives: directives}).Parse()
		return err
	}

	if err := parseWith("struct Point { x, y };"); err != nil {
		t.Fatalf("declaring: %v", err)
	}
	if err := parseWith("ident p = Point{10, 20};"); err != nil {
		t.Fatalf("building on a later line: %v", err)
	}
	if err := parseWith("printd p.y;"); err != nil {
		t.Fatalf("reading a field on a later line: %v", err)
	}
}
