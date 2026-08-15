package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
)

// Reading a field is resolved while parsing: the name becomes the index it was declared at,
// and the tree carries the index.
func TestFieldResolvesToItsIndex(t *testing.T) {
	cases := []struct {
		source string
		want   uint64
	}{
		{source: "struct Point { x, y };\nident p = Point(1, 2);\np.x;", want: 0},
		{source: "struct Point { x, y };\nident p = Point(1, 2);\np.y;", want: 1},
		{source: "struct Row { a, b, c };\nident r = Row(1, 2, 3);\nr.c;", want: 2},
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

// A declared struct in front of parentheses builds a value; anything else applies values to
// a scope. The two look the same and are told apart by the directive.
func TestStructConstructionIsNotACall(t *testing.T) {
	nodes := parse(t, "struct Point { x, y };\nPoint(1, 2);")
	if _, ok := nodes[1].(StructLiteral); !ok {
		t.Errorf("Point(1, 2) parsed as %T, want a StructLiteral", nodes[1])
	}

	nodes = parse(t, "ident greet = defer { 1; };\ngreet();")
	if _, ok := nodes[1].(CalleeLiteral); !ok {
		t.Errorf("greet() parsed as %T, want a CalleeLiteral", nodes[1])
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
			source: "struct Point { x, y };\nident p = Point(1, 2);\np.z;",
			want:   "struct Point has no field named z",
		},
		{
			name:   "value with no shape",
			source: "struct Point { x, y };\nident p = 1;\np.x;",
			want:   "nothing says which struct this value is",
		},
		{
			name:   "too few values",
			source: "struct Point { x, y };\nPoint(1);",
			want:   "struct Point has 2 fields (x, y) but got 1",
		},
		{
			name:   "too many values",
			source: "struct Point { x, y };\nPoint(1, 2, 3);",
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
			positioned, ok := err.(*lexer.Error)
			if !ok {
				t.Fatalf("error is %T, want a positioned *lexer.Error", err)
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
	_, err := parseSource(t, "struct Point { x, y };\nident p = Point(1, 2);\np.x.y;", "main.ar")
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
		"struct Point { x, y };\nident p = Point(1, 2);\nprintd p + 1;",
		"struct Point { x, y };\nprintd Point(1, 2) equals Point(1, 2);",
		"struct Point { x, y };\nident p = Point(1, 2);\nident q = p;\nprintd q.y;",
	} {
		if _, err := parseSource(t, source, "main.ar"); err != nil {
			t.Errorf("parsing %q: %v", source, err)
		}
	}
}
