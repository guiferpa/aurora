package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// What a block promises, and what happens when it does not keep it.
//
// `as` is a claim the compiler believes; this is a promise it checks. The two live in one
// program, and a block that promises nothing goes on answering with a run of tapes.

const person = "shape Person { name };\n"
const result = "shape Result { failed, value };\n"

// The promise is read at the end of a block, and it is the same place for a deferred scope,
// which is a block with a word in front of it.
func TestABlockPromisesAShape(t *testing.T) {
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
			// The promise is kept by a name whose shape was named, the same way a field read
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

// The promise reaches the tree, so whoever reads the block knows what it answers with.
func TestThePromiseIsOnTheBlock(t *testing.T) {
	nodes := parse(t, person+"{ Person{\"Joana\"}; } returns Person;")

	block, ok := nodes[1].(ast.BlockExpression)
	if !ok {
		t.Fatalf("the second node is %T, want a block", nodes[1])
	}
	if block.Returns != "Person" {
		t.Errorf("the block promises %q, want Person", block.Returns)
	}
}

// A block that promises nothing is a block, and answers with a run of tapes as it always did.
func TestABlockWithoutAPromiseIsUnchanged(t *testing.T) {
	nodes := parse(t, person+"{ Person{\"Joana\"}; };")

	if block := nodes[1].(ast.BlockExpression); block.Returns != "" {
		t.Errorf("the block promises %q, want nothing", block.Returns)
	}
}

// Every way of breaking the promise, and what it says.
var brokenPromises = []struct {
	name   string
	source string
	want   string
}{
	{
		name:   "ends with a number",
		source: person + "{ ident p = Person{\"Joana\"};\n10; } returns Person;",
		want:   "answers with Person and ends with a number",
	},
	{
		name:   "ends with another shape",
		source: person + "shape Place { city };\n{ Place{\"Recife\"}; } returns Person;",
		want:   "answers with Person and ends with a Place",
	},
	{
		name:   "an if with no else",
		source: person + "{ if true { Person{\"Joana\"}; }; } returns Person;",
		want:   "its if has no else",
	},
	{
		name:   "an else that answers with something else",
		source: person + "{ if true { Person{\"Joana\"}; } else { 0; }; } returns Person;",
		want:   "the else answers with a number",
	},
	{
		name:   "an if arm that answers with something else",
		source: person + "{ if true { 0; } else { Person{\"Joana\"}; }; } returns Person;",
		want:   "the if answers with a number",
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

func TestABlockThatDoesNotKeepItsPromise(t *testing.T) {
	for _, tc := range brokenPromises {
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

// A scope that promised gives its callers a shape, so the claim disappears from the place it
// used to be repeated once per call.
func TestACallToAScopeThatPromisedHasAShape(t *testing.T) {
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

// A scope that promised nothing gives nothing away, and reading a field off it still asks for
// the claim — which is the half of the rule that does not change.
func TestACallToAScopeThatPromisedNothing(t *testing.T) {
	const source = result + "ident divide = defer { Result{0, 5}; };\nident r = divide(10, 2);\nprintd r.value;"

	_, err := parseSource(t, source, "main.ar")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "nothing says which shape this value is") {
		t.Errorf("error = %q", err)
	}
}

// A block bound to a name is the run of tapes itself, so the promise is the name's own shape.
func TestABlockBoundToANameCarriesItsPromise(t *testing.T) {
	if _, err := parseSource(t, person+"ident p = { Person{\"Joana\"}; } returns Person;\nprintc p.name;", "main.ar"); err != nil {
		t.Errorf("parsing: %v", err)
	}
}
