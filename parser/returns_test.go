package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// What a block declares it returns, and what happens when it does not keep it.
//
// `as` is a claim the compiler believes; this is a declaration it checks. The two live in one
// program, and a block that declares nothing goes on returning a run of tapes.

const person = "shape Person { name };\n"
const result = "shape Result { failed, value };\n"

// The `returns` is read at the end of a block, and it is the same place for a deferred scope,
// which is a block with a word in front of it.
func TestABlockDeclaresWhatItReturns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			name:   "a block",
			source: person + "{ ident p = Person{\"Joana\"};\np; } returns Person;",
		},
		{
			name:   "a deferred scope",
			source: person + "ident make = defer {\nPerson{\"Joana\"};\n} returns Person;",
		},
		{
			name: "an if with both arms agreeing",
			source: result + "ident divide = defer {\n" +
				"if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; };\n" +
				"} returns Result;",
		},
		{
			// A branch is nested ifs by the time anything reads the tree, and its last item
			// is the innermost else — so the way out is always covered.
			name: "a branch",
			source: result + "ident classify = defer {\n" +
				"branch { feed(0) equals 0: Result{0, 100}, feed(0) equals 1: Result{0, 200}, Result{1, 0}; };\n" +
				"} returns Result;",
		},
		{
			// It is kept by a name whose shape was named, the same way a field read
			// is allowed by one.
			name:   "a name that was shaped",
			source: person + "ident make = defer {\nident p = feed(0) as Person;\np;\n} returns Person;",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseSource(t, tc.source, "main.ar"); err != nil {
				t.Fatalf("parsing: %v", err)
			}
		})
	}
}

// It reaches the tree, so whoever reads the block knows what it returns.
func TestWhatItReturnsIsOnTheBlock(t *testing.T) {
	nodes := parse(t, person+"{ Person{\"Joana\"}; } returns Person;")

	block, ok := nodes[1].(ast.BlockExpression)
	if !ok {
		t.Fatalf("the second node is %T, want a block", nodes[1])
	}
	if block.Returns != "Person" {
		t.Errorf("the block returns %q, want Person", block.Returns)
	}
}

// A block that declares nothing is a block, and returns a run of tapes as it always did.
func TestABlockWithoutADeclarationIsUnchanged(t *testing.T) {
	nodes := parse(t, person+"{ Person{\"Joana\"}; };")

	if block := nodes[1].(ast.BlockExpression); block.Returns != "" {
		t.Errorf("the block returns %q, want nothing", block.Returns)
	}
}

// Every way of breaking a declaration, and what it says.
var brokenDeclarations = []struct {
	name   string
	source string
	want   string
}{
	{
		name:   "ends with a number",
		source: person + "{ ident p = Person{\"Joana\"};\n10; } returns Person;",
		want:   "returns Person and ends with a number",
	},
	{
		name:   "ends with another shape",
		source: person + "shape Place { city };\n{ Place{\"Recife\"}; } returns Person;",
		want:   "returns Person and ends with a Place",
	},
	{
		name:   "an if with no else",
		source: person + "{ if true { Person{\"Joana\"}; }; } returns Person;",
		want:   "its if has no else",
	},
	{
		name:   "an else that returns something else",
		source: person + "{ if true { Person{\"Joana\"}; } else { 0; }; } returns Person;",
		want:   "the else returns a number",
	},
	{
		name:   "an if arm that returns something else",
		source: person + "{ if true { 0; } else { Person{\"Joana\"}; }; } returns Person;",
		want:   "the if returns a number",
	},
	{
		// Deeper than one arm. What a body returns is decided in one place, and this is the
		// other walk — the one that only runs once it is already broken, to say where. It has
		// to keep descending, or a break far enough in would be reported as the whole if.
		name:   "an arm of an inner if that returns something else",
		source: person + "{ if true { if true { 0; } else { Person{\"Joana\"}; }; } else { Person{\"Joana\"}; }; } returns Person;",
		want:   "the if returns a number",
	},
	{
		name:   "a shape nobody declared",
		source: "{ 10; } returns Person;",
		want:   "Person is not a declared shape",
	},
	{
		name:   "an empty block",
		source: person + "{ } returns Person;",
		want:   "ends with nothing",
	},
}

func TestABlockThatDoesNotKeepItsDeclaration(t *testing.T) {
	for _, tc := range brokenDeclarations {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSource(t, tc.source, "main.ar")
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to say %q", err, tc.want)
			}
			positioned, ok := err.(*token.Error)
			if !ok || positioned.Line == 0 {
				t.Errorf("error is %T and unpositioned", err)
			}
		})
	}
}

// A scope that declared gives its callers a shape, so the claim disappears from the place it
// used to be repeated once per call.
func TestACallToAScopeThatDeclaredHasAShape(t *testing.T) {
	const source = result +
		"ident divide = defer {\n" +
		"if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; };\n" +
		"} returns Result;\n"

	for _, tc := range []struct{ name, read string }{
		{name: "bound and then read", read: "ident r = divide(10, 2);\nprintd r.value;"},
		{name: "read straight off the call", read: "printd divide(10, 2).value;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseSource(t, source+tc.read, "main.ar"); err != nil {
				t.Errorf("parsing: %v", err)
			}
		})
	}
}

// A scope that wrote no `returns` is understood anyway, because the compiler reads what its
// body ends with — which is the same walk that checks a `returns` when there is one.
//
// So the claim disappears from the caller without the scope having to declare anything. What
// `returns` is for is what a walk cannot reach.
func TestACallToAScopeThatDeclaredNothing(t *testing.T) {
	const source = result + "ident divide = defer { Result{0, 5}; };\nident r = divide(10, 2);\nprintd r.value;"

	if _, err := parseSource(t, source, "main.ar"); err != nil {
		t.Errorf("parsing: %v", err)
	}
}

// What is still not known is a body that ends with something that is not a shape, and reading
// a field off it is refused where it was written — which is the half of the rule that does not
// change, and the reason the message is still worth having.
func TestACallToAScopeWhoseShapeNothingSays(t *testing.T) {
	const source = result + "ident divide = defer { feed(0) / feed(1); };\nident r = divide(10, 2);\nprintd r.value;"

	_, err := parseSource(t, source, "main.ar")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "nothing says which shape this value is") {
		t.Errorf("error = %q", err)
	}
}

// A block bound to a name is the run of tapes itself, so what it returns is the name's own shape.
func TestABlockBoundToANameCarriesWhatItReturns(t *testing.T) {
	if _, err := parseSource(t, person+"ident p = { Person{\"Joana\"}; } returns Person;\nprintc p.name;", "main.ar"); err != nil {
		t.Errorf("parsing: %v", err)
	}
}

// What a scope returns says who worked it out, which is the only difference between the two
// ways of knowing.
//
// It exists so somebody can be shown what the compiler knows and where it got it — an editor
// telling a shape apart from a claim, a message about a `returns` that was not kept. The shape
// itself is used the same either way, which is the point of inferring it.
func TestWhatAScopeReturnsSaysWhoSaidIt(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   string
		declared bool
	}{
		{
			name:     "the file wrote it",
			source:   person + "ident make = defer { Person{\"Joana\"}; } returns Person;",
			declared: true,
		},
		{
			name:     "the compiler read the body",
			source:   person + "ident make = defer { Person{\"Joana\"}; };",
			declared: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := parseSource(t, tc.source, "main.ar")
			if err != nil {
				t.Fatalf("parsing: %v", err)
			}
			if len(tree.Returns) != 1 {
				t.Fatalf("the tree carries %d, want the one scope", len(tree.Returns))
			}
			returns := tree.Returns[0]
			if returns.Scope != "make" || returns.Shape != "Person" {
				t.Errorf("%s returns %q, want make returning Person", returns.Scope, returns.Shape)
			}
			if returns.Declared != tc.declared {
				t.Errorf("declared = %v, want %v", returns.Declared, tc.declared)
			}
		})
	}
}
