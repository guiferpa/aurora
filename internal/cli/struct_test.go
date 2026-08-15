package cli

import (
	"strings"
	"testing"
)

// A struct value is a run of tapes, one per field, and nothing else. These check it at the
// answer, which is the only place the shape can be observed at all.
func TestStructReadsItsFields(t *testing.T) {
	const source = "struct Point { x, y };\nident p = Point{10, 20};\n"

	cases := []struct {
		source string
		want   string
	}{
		{source: source + "printd p.x;", want: "10"},
		{source: source + "printd p.y;", want: "20"},
		// printd walks tapes, and a struct is tapes, so it reads the whole thing without
		// being taught anything about structs.
		{source: source + "printd p;", want: "10 20"},
		{source: source + "printb p;", want: "[0 0 0 0 0 0 0 10 0 0 0 0 0 0 0 20]"},
	}

	for _, tc := range cases {
		t.Run(strings.ReplaceAll(tc.source, "\n", " "), func(t *testing.T) {
			got := output(t, tc.source, 0)
			if got[len(got)-1] != tc.want {
				t.Errorf("printed %s, want %s", got[len(got)-1], tc.want)
			}
		})
	}
}

// The invariant the whole design rests on: the directive dies at compile time. A struct is
// the same bytes as a reel of the same length, because it is one — so a program that builds
// a struct and a program that builds the equivalent reel answer identically.
func TestStructIsIndistinguishableFromAReel(t *testing.T) {
	// "ab" is two characters, so the reel is two tapes holding 97 and 98.
	viaStruct := output(t, "struct Pair { a, b };\nprintb Pair{97, 98};", 0)[0]
	viaReel := output(t, "printb \"ab\";", 0)[0]

	if viaStruct != viaReel {
		t.Errorf("struct wrote %s, reel wrote %s — they must be the same bytes", viaStruct, viaReel)
	}
}

// Two structs of the same width are the same value, which follows from there being nothing
// in the bytes that says which struct they came from.
func TestStructsOfTheSameWidthAreEqual(t *testing.T) {
	source := "struct Point { x, y };\nstruct Pair { a, b };\nprintd Point{1, 2} equals Pair{1, 2};"
	if got := output(t, source, 0)[0]; got != "1" {
		t.Errorf("comparing two structs of the same bytes gave %s, want 1", got)
	}
}

// Passing a struct to a scope is the point of having one: behaviour lives in defer, and a
// defer receives values through feed. The shape is lost crossing that line, so it is named
// again with `as`.
func TestStructThroughAScope(t *testing.T) {
	source := `struct Point { x, y };

ident area = defer {
  ident p = feed(0) as Point;
  p.x * p.y;
};

printd area(Point{10, 20});`

	if got := output(t, source, 0)[0]; got != "200" {
		t.Errorf("area printed %s, want 200", got)
	}
}

// The annotation is an expression, so it does not need a name to hang on.
func TestShapeAnnotatedInline(t *testing.T) {
	source := "struct Point { x, y };\nident twice = defer { (feed(0) as Point).y * 2; };\nprintd twice(Point{3, 7});"
	if got := output(t, source, 0)[0]; got != "14" {
		t.Errorf("printed %s, want 14", got)
	}
}

// A field is a tape, and a tape holding a scope index is what a defer is, so a struct
// carries one through unharmed.
//
// Calling it back is another matter and does not work: OpCall resolves a name in the
// environ, not a value, so there is nothing to apply a tape to. A field holds the scope,
// it does not call it.
func TestStructFieldHoldingADefer(t *testing.T) {
	source := `struct Op { run, value };

ident double = defer { feed(0) * 2; };
ident op = Op{double, 21};

printd double;
printd op.run;`

	got := output(t, source, 0)
	if got[0] != got[1] {
		t.Errorf("the field gave %s but the defer is %s — a field has to carry it unchanged", got[1], got[0])
	}
}

// The offset of a field is its index times the tape width, so the same source answers the
// same at any width.
func TestStructAcrossTapeSizes(t *testing.T) {
	const source = "struct Point { x, y };\nident p = Point{10, 20};\nprintd p.x;\nprintd p.y;"

	for _, tapeSize := range []int{1, 2, 8, 32} {
		got := output(t, source, tapeSize)
		if got[0] != "10" || got[1] != "20" {
			t.Errorf("%d-byte tapes: got %v, want [10 20]", tapeSize, got)
		}
	}
}

// Reading past the end gives the neutral value rather than stopping the program, the way
// head saturates and feed wraps. The annotation is a claim, and a claim can be wrong.
func TestFieldPastTheEndIsNeutral(t *testing.T) {
	source := "struct Point { x, y };\nident p = 7 as Point;\nprintd p.x;\nprintd p.y;"
	got := output(t, source, 0)
	if got[0] != "7" {
		t.Errorf("first field of a one-tape value gave %s, want 7", got[0])
	}
	if got[1] != "0" {
		t.Errorf("second field gave %s, want the neutral value", got[1])
	}
}

// A field is one tape wide, and one character is one tape, so a reel of one fits.
func TestAOneCharacterReelFitsAField(t *testing.T) {
	source := "struct Pair { a, b };\nprintd Pair{\"a\", 1};"
	if got := output(t, source, 0)[0]; got != "97 1" {
		t.Errorf("printed %s, want \"97 1\"", got)
	}
}
