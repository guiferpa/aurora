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

// A scope has no arity, but a body says how many positions it can address: the highest index
// it feeds, plus one. Applying fewer is not an error — what was not applied answers with a
// tape of zeros, which is a defined answer — but that answer is silent, so it is said here.
func TestApplyingFewerValuesThanTheScopeReads(t *testing.T) {
	source := "ident sum = defer { feed(0) + feed(1); };\nprintb sum(5);\n"

	warnings := warningsFor(t, source, 0)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %v", warnings)
	}
	message := warnings[0].Message
	for _, want := range []string{"sum", "2 positions", "1 were applied", "feed(1)"} {
		if !strings.Contains(message, want) {
			t.Errorf("warning = %q, want it to mention %q", message, want)
		}
	}
	// It points at the call, not at the scope: the scope is fine, and the call is what has
	// to change.
	if warnings[0].Line != 2 {
		t.Errorf("warning points at line %d, want the line of the call", warnings[0].Line)
	}
}

// The highest index is what counts, not how many are read: a body that only reads feed(3)
// still needs four positions for that read to reach anything.
func TestTheHighestPositionIsWhatCounts(t *testing.T) {
	source := "ident third = defer { feed(3); };\nprintb third(1, 2, 3);\n"

	warnings := warningsFor(t, source, 0)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0].Message, "4 positions") {
		t.Errorf("warning = %q, want it to count four positions", warnings[0].Message)
	}
}

func TestNoWarningWhenEnoughWasApplied(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "exactly what it reads",
			source: "ident sum = defer { feed(0) + feed(1); };\nprintb sum(1, 2);",
		},
		{
			// Extra values are unread, and that is the language: a scope is blind to how
			// many arrive.
			name:   "more than it reads",
			source: "ident double = defer { feed(0) * 2; };\nprintb double(5, 99, 100);",
		},
		{
			name:   "a scope that reads nothing",
			source: "ident two = defer { 2; };\nprintb two();",
		},
		{
			// The feeds of a nested scope read the vector applied to it, not the one
			// applied here.
			name:   "a nested scope reading further",
			source: "ident outer = defer { ident inner = defer { feed(3); }; feed(0); };\nprintb outer(1);",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if warnings := warningsFor(t, tc.source, 0); len(warnings) != 0 {
				t.Errorf("expected no warning, got %v", warnings)
			}
		})
	}
}

// It stays quiet unless it is sure. Which body a call reaches is a runtime question whenever
// the name does not answer it on its own, and a warning that guesses is worse than none.
func TestItSaysNothingWhenTheScopeIsNotKnown(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "an alias, which is a name bound to a name",
			source: "ident sum = defer { feed(0) + feed(1); };\nident f = sum;\nprintb f(5);",
		},
		{
			name:   "a name bound twice, to scopes reading different amounts",
			source: "ident f = defer { feed(0) + feed(1); };\nident f = defer { feed(0); };\nprintb f(5);",
		},
		{
			name:   "a name answered by a branch",
			source: "ident a = defer { feed(0) + feed(1); };\nident b = defer { feed(0); };\nident g = if true { a; } else { b; };\nprintb g(5);",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if warnings := warningsFor(t, tc.source, 0); len(warnings) != 0 {
				t.Errorf("expected no warning, got %v", warnings)
			}
		})
	}
}

// A call written inside an operand, inside the test of an if or as another call's argument is
// still a call. This is what the walk is for, and what it used to miss.
func TestACallIsFoundWhereverItIsWritten(t *testing.T) {
	const scope = "ident sum = defer { feed(0) + feed(1); };\n"

	cases := []struct {
		name   string
		source string
	}{
		{name: "in an operand", source: scope + "printb 1 + sum(5);"},
		{name: "in the test of an if", source: scope + "printb if sum(5) bigger 1 { 1; };"},
		{name: "as an argument", source: scope + "printb sum(sum(5), 2);"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if warnings := warningsFor(t, tc.source, 0); len(warnings) != 1 {
				t.Errorf("expected one warning, got %v", warnings)
			}
		})
	}
}
