package emitter

import (
	"strconv"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/diag"
)

func warningsFor(t *testing.T, source string, tapeSize int) []diag.Warning {
	t.Helper()
	return compileWith(t, source, tapeSize).Warnings
}

// A scope holding more deferred scopes than its tape can index gets a warning, because the
// index wraps and a call reaches a different scope instead of failing. It is a warning and
// not an error: the count is static and cannot know how often a scope runs, and a program
// already running should never be stopped by it.
func TestDeferCapacityWarning(t *testing.T) {
	source := strings.Repeat("ident d = defer { 1; };\n", 300)
	// Every binding is named the same, which the evaluator would reject — the warning is a
	// compile-time count and does not care.

	warnings := warningsFor(t, source, 1)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %v", warnings)
	}
	message := warnings[0].Message
	for _, want := range []string{"300", "1-byte", "256"} {
		if !strings.Contains(message, want) {
			t.Errorf("warning = %q, want it to mention %q", message, want)
		}
	}
}

func TestNoWarningWhenTheyFit(t *testing.T) {
	cases := []struct {
		name     string
		defers   int
		tapeSize int
	}{
		{name: "few defers, one byte", defers: 10, tapeSize: 1},
		{name: "exactly the capacity", defers: 256, tapeSize: 1},
		{name: "many defers, two bytes", defers: 300, tapeSize: 2},
		{name: "many defers, default tape", defers: 300, tapeSize: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Repeat("ident d = defer { 1; };\n", tc.defers)
			if warnings := warningsFor(t, source, tc.tapeSize); len(warnings) != 0 {
				t.Errorf("expected no warnings, got %v", warnings)
			}
		})
	}
}

// The tally is per scope, since each running scope keeps its own. Defers spread across
// nested scopes do not add up into one.
func TestDeferCapacityCountsPerScope(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 4; i++ {
		source.WriteString("{\n")
		source.WriteString(strings.Repeat("ident d = defer { 1; };\n", 100))
		source.WriteString("};\n")
	}

	if warnings := warningsFor(t, source.String(), 1); len(warnings) != 0 {
		t.Errorf("400 defers across four scopes fit: %v", warnings)
	}
}

func TestDeferCapacityWarnsInsideANestedScope(t *testing.T) {
	source := "{\n" + strings.Repeat("ident d = defer { 1; };\n", 300) + "};\n"

	if warnings := warningsFor(t, source, 1); len(warnings) != 1 {
		t.Errorf("expected the inner scope to be counted, got %v", warnings)
	}
}

func TestWarningsAreReportedPerTapeSize(t *testing.T) {
	source := strings.Repeat("ident d = defer { 1; };\n", 70000)
	// 70000 exceeds a two-byte tape (65536) but fits anything wider.
	for _, size := range []int{1, 2} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			if warnings := warningsFor(t, source, size); len(warnings) == 0 {
				t.Error("expected a warning")
			}
		})
	}
	if warnings := warningsFor(t, source, 4); len(warnings) != 0 {
		t.Errorf("a four-byte tape names plenty: %v", warnings)
	}
}

// treeOf parses source and answers the nodes of it, so a walk can be followed from the top.
//
// The filename is a parameter because the parser reads it: assert is only accepted in a
// ".test.ar" file, and that is where the warning about it has anything to say.
func treeOf(t *testing.T, filename, source string) []ast.Node {
	t.Helper()
	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: filename, Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	return tree.Nodes
}

// reaches answers whether a full walk from the top finds a feed, which is the probe: it is a
// leaf expression and can be written almost anywhere one is allowed.
func reaches(nodes []ast.Node) bool {
	found := false
	var walk func([]ast.Node)
	walk = func(scope []ast.Node) {
		for _, node := range scope {
			if _, ok := node.(ast.FeedExpression); ok {
				found = true
			}
			walk(childScopesOf(node))
		}
	}
	walk(nodes)
	return found
}

// A walk that does not reach an expression does not fail — it goes quiet, and a check built on
// it answers "nothing to say" for a program it never looked at. Every place an expression can
// be written has to be reachable, so each of these is one that used to be a blind spot.
func TestTheWalkReachesEveryExpression(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "an operand of an operator", source: "printb 1 + feed(7);"},
		{name: "an operand of a comparison", source: "printb feed(7) bigger 1;"},
		{name: "an operand of and/or", source: "printb feed(7) and true;"},
		{name: "the test of an if", source: "printb if feed(7) bigger 1 { 1; };"},
		{name: "an argument of a call", source: "ident f = defer { 1; };\nprintb f(feed(7));"},
		{name: "an item of a tape", source: "printb [feed(7)];"},
		// pull, push, head and tail are walked too, and are not probed here: the parser
		// narrows what may sit in those positions (isValidTapeTarget, isValidTapeItem), so
		// a feed cannot be written there to look for.
		{name: "the operand of a negation", source: "printb -feed(7);"},
		{name: "inside parentheses", source: "printb (feed(7));"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !reaches(treeOf(t, "main.ar", tc.source)) {
				t.Errorf("the walk did not reach the feed in %q", tc.source)
			}
		})
	}
}

// And the gap seen from where it is felt. An assert written as the argument of a call is an
// assert like any other: it runs under "aurora test" and is consumed in silence in a plain
// run, which is exactly what the warning exists to say out loud — and it used to say nothing,
// because the walk stopped at the call.
func TestAnAssertIsFoundWhereverItIsWritten(t *testing.T) {
	source := "ident f = defer { 1; };\nprintb f(assert(1 equals 1, \"one\"));\n"

	warnings := checkAsserts(treeOf(t, "main.test.ar", source))
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0].Message, "assert only runs under") {
		t.Errorf("warning = %q, want it to be about the assert", warnings[0].Message)
	}
	if warnings[0].Line != 2 {
		t.Errorf("warning points at line %d, want the line the assert is on", warnings[0].Line)
	}
}
