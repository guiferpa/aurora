package cli

import (
	"strings"
	"testing"
)

// A shape value is a run of tapes, one per field, and nothing else. These check it at the
// answer, which is the only place the shape can be observed at all.
func TestShapeReadsItsFields(t *testing.T) {
	const source = "shape Point { x, y };\nident p = Point{10, 20};\n"

	cases := []struct {
		source string
		want   string
	}{
		{source: source + "printd p.x;", want: "10"},
		{source: source + "printd p.y;", want: "20"},
		// printd walks tapes, and a shape is tapes, so it reads the whole thing without
		// being taught anything about shapes.
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

// The invariant the whole design rests on: the declaration dies at compile time. A shape
// value is a run of tapes and carries nothing that says a shape built it, so two shapes
// of the same width are one value — checked below — and the tape at index 0 is just a tape.
func TestShapeCarriesNothingOfTheDeclaration(t *testing.T) {
	// A field read out of a shape is the same value written on its own.
	fromShape := output(t, "shape Pair { a, b };\nprintb Pair{97, 98}.a;", 0)[0]
	onItsOwn := output(t, "printb 97;", 0)[0]

	if fromShape != onItsOwn {
		t.Errorf("a field wrote %s, the value alone wrote %s — a field is just a tape", fromShape, onItsOwn)
	}
}

// Two shapes of the same width are the same value, which follows from there being nothing
// in the bytes that says which shape they came from.
func TestShapesOfTheSameWidthAreEqual(t *testing.T) {
	source := "shape Point { x, y };\nshape Pair { a, b };\nprintd Point{1, 2} equals Pair{1, 2};"
	if got := output(t, source, 0)[0]; got != "1" {
		t.Errorf("comparing two shapes of the same bytes gave %s, want 1", got)
	}
}

// Passing a shape to a scope is the point of having one: behaviour lives in defer, and a
// defer receives values through feed. The shape is lost crossing that line, so it is named
// again with `as`.
func TestShapeThroughAScope(t *testing.T) {
	source := `shape Point { x, y };

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
	source := "shape Point { x, y };\nident twice = defer { (feed(0) as Point).y * 2; };\nprintd twice(Point{3, 7});"
	if got := output(t, source, 0)[0]; got != "14" {
		t.Errorf("printed %s, want 14", got)
	}
}

// A field is a tape, and a tape holding a scope index is what a defer is, so a shape
// carries one through unharmed.
//
// Calling it back is another matter and does not work: OpCall resolves a name in the
// environ, not a value, so there is nothing to apply a tape to. A field holds the scope,
// it does not call it.
func TestShapeFieldHoldingADefer(t *testing.T) {
	source := `shape Op { run, value };

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
func TestShapeAcrossTapeSizes(t *testing.T) {
	const source = "shape Point { x, y };\nident p = Point{10, 20};\nprintd p.x;\nprintd p.y;"

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
	source := "shape Point { x, y };\nident p = 7 as Point;\nprintd p.x;\nprintd p.y;"
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
	source := "shape Pair { a, b };\nprintd Pair{\"a\", 1};"
	if got := output(t, source, 0)[0]; got != "97 1" {
		t.Errorf("printed %s, want \"97 1\"", got)
	}
}

// A shape crossing a deferred scope, which is what a composite answer is made of: a scope
// that has to say two things — whether it worked and what it produced — says them as one run
// of tapes, and whoever called it reads them back by naming the shape.
func TestAShapeComesOutOfAScope(t *testing.T) {
	const declaration = "shape Result { failed, value };\n"

	cases := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "built inside and read outside",
			source: declaration + "ident make = defer { Result{1, 42}; };\nident r = make() as Result;\nprintd r.failed;\nprintd r.value;",
			want:   []string{"1", "42"},
		},
		{
			name: "chosen by a branch",
			source: declaration +
				"ident divide = defer { if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; }; };\n" +
				"ident ok = divide(10, 2) as Result;\nident bad = divide(10, 0) as Result;\n" +
				"printd ok.failed;\nprintd ok.value;\nprintd bad.failed;",
			want: []string{"0", "5", "1"},
		},
		{
			// The shape binds tighter than anything, so it can be named and read through in
			// one expression, without a name in between.
			name:   "shaped and read in one expression",
			source: declaration + "ident make = defer { Result{1, 42}; };\nprintd make() as Result.value;",
			want:   []string{"42"},
		},
		{
			name: "fed into another scope",
			source: declaration +
				"ident make = defer { Result{0, 7}; };\n" +
				"ident twice = defer { ident r = feed(0) as Result; r.value * 2; };\n" +
				"printd twice(make());",
			want: []string{"14"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := output(t, tc.source, 0)
			if len(got) != len(tc.want) {
				t.Fatalf("printed %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("printed %v, want %v", got, tc.want)
					break
				}
			}
		})
	}
}

// A promise, kept, and the claim gone from the call site: the shape is known because the
// scope said so, and the compiler checked that it said the truth.
func TestAPromisedShapeIsReadWithoutAClaim(t *testing.T) {
	const source = "shape Result { failed, value };\n" +
		"ident divide = defer {\n" +
		"  if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; };\n" +
		"} returns Result;\n" +
		"ident r = divide(10, 2);\n" +
		"printd r.failed;\nprintd r.value;\nprintd divide(1, 0).failed;"

	got := output(t, source, 0)
	want := []string{"0", "5", "1"}
	if len(got) != len(want) {
		t.Fatalf("printed %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("printed %v, want %v", got, want)
		}
	}
}
