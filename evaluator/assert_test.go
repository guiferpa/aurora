package evaluator

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// evaluateAsserts compiles source as a test file and runs it, with assertions on or off.
func evaluateAsserts(t *testing.T, source string, asserts bool) *Evaluator {
	t.Helper()

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "checks.test.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	ev := New(NewEvaluatorOptions{Asserts: asserts})
	if _, err := ev.Evaluate(insts); err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	return ev
}

// A run collects every assertion, not only the failures, so a report can say how many held.
func TestAssertResultsRecordWhatPassed(t *testing.T) {
	ev := evaluateAsserts(t, `assert(1 equals 1, "first holds");
assert(1 equals 2, "second does not");
assert(2 bigger 1, "third holds");
`, true)

	results := ev.GetAssertResults()
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	want := []struct {
		passed  bool
		message string
	}{
		{passed: true, message: "first holds"},
		{passed: false, message: "second does not"},
		{passed: true, message: "third holds"},
	}
	for i, tc := range want {
		if results[i].Passed != tc.passed {
			t.Errorf("result %d passed = %v, want %v", i, results[i].Passed, tc.passed)
		}
		if !strings.Contains(results[i].Message, tc.message) {
			t.Errorf("result %d message = %q, want it to carry %q", i, results[i].Message, tc.message)
		}
	}

	// The failures are what a plain run reports.
	errs := ev.GetAssertErrors()
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "second does not") {
		t.Errorf("error = %q, want the failing message", errs[0])
	}
}

// Assertions belong to a runner that asked for them. A plain run consumes the operands and
// moves on, so a program holding an assertion that would fail is not affected by it.
func TestAssertionsAreIgnoredWhenNotAsked(t *testing.T) {
	ev := evaluateAsserts(t, `assert(1 equals 2, "would fail");
assert(1 equals 1, "would pass");
`, false)

	if results := ev.GetAssertResults(); len(results) != 0 {
		t.Errorf("got %d results, want none: %+v", len(results), results)
	}
	if errs := ev.GetAssertErrors(); len(errs) != 0 {
		t.Errorf("got %d errors, want none: %v", len(errs), errs)
	}
}

// An assertion is an expression like any other, so it leaves a value behind whether or not
// it was evaluated.
func TestAssertionProducesTheNeutralValue(t *testing.T) {
	for _, asserts := range []bool{true, false} {
		ev := evaluateAsserts(t, `assert(1 equals 1, "holds");`, asserts)
		if len(ev.environ.GetTemps()) == 0 {
			t.Errorf("asserts=%v: the assertion left no value behind", asserts)
		}
	}
}

// Two programs run in one evaluator each number their labels from zero, so what the first
// left behind sits under the labels the second is about to use.
func TestClearTemps(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), []byte{1, 2, 3})

	if len(ev.environ.GetTemps()) == 0 {
		t.Fatal("the temp should be there before clearing")
	}
	ev.ClearTemps()
	if got := len(ev.environ.GetTemps()); got != 0 {
		t.Errorf("%d temps survived the clear", got)
	}
}
