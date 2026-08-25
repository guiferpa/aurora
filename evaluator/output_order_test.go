package evaluator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// tagged is a printer that collects what it was asked to print, tagged, so the order of a
// whole session can be compared against what the source says should happen.
type tagged struct {
	tag   string
	lines *[]string
}

func (t tagged) Print(value []byte) ([]byte, error) {
	*t.lines = append(*t.lines, fmt.Sprintf("%s %v", t.tag, value))
	return value, nil
}

// runInOrder compiles source and runs one top-level expression at a time, recording what
// each one printed and the value it produced — the loop a host like the playground runs.
func runInOrder(t *testing.T, source string) []string {
	t.Helper()

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	program, err := emitter.New(emitter.NewEmitterOptions{}).EmitProgram(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	lines := make([]string, 0)
	ev := New(NewEvaluatorOptions{
		PrintBytes: tagged{tag: "printb", lines: &lines},
	})

	for at, expr := range program.Expressions {
		var until func(ir.Point) bool
		if at+1 < len(program.Expressions) {
			next := program.Expressions[at+1].At
			until = func(point ir.Point) bool { return point == next }
		}
		temps, err := ev.EvaluateBlocks(program.Blocks, expr.At, until, "")
		if err != nil {
			t.Fatalf("evaluating: %v", err)
		}
		if value, ok := temps[byteutil.ToHex(expr.Label)]; ok {
			lines = append(lines, fmt.Sprintf("value %v", value))
		}
	}
	return lines
}

// Issue #11: output came out in the wrong order because every printb was written while the
// program ran and the values only afterwards, walked out of a map. Running one expression
// at a time puts each value next to the prints that came with it.
func TestOutputFollowsSourceOrder(t *testing.T) {
	got := runInOrder(t, `if 10 bigger 9 {
  10;
};

if 11 bigger 10 {
  printb 20;
};
`)

	want := []string{
		"value [0 0 0 0 0 0 0 10]",  // the first if produced 10
		"printb [0 0 0 0 0 0 0 20]", // then the second one printed 20
		"value [0 0 0 0 0 0 0 20]",  // and is worth what it printed
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("output order:\ngot:\n  %s\nwant:\n  %s",
			strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// A print is an expression like any other, so it is worth something: the value it showed.
// It used to be worth nothing — an effect with no temp behind it — and an expression worth
// nothing has nothing to say to the one that reads it.
func TestOutputInterleavesPrintsAndValues(t *testing.T) {
	got := runInOrder(t, "printb 1;\n2;\nprintb 3;\n4;\n")

	want := []string{
		"printb [0 0 0 0 0 0 0 1]",
		"value [0 0 0 0 0 0 0 1]",
		"value [0 0 0 0 0 0 0 2]",
		"printb [0 0 0 0 0 0 0 3]",
		"value [0 0 0 0 0 0 0 3]",
		"value [0 0 0 0 0 0 0 4]",
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("output order:\ngot:\n  %s\nwant:\n  %s",
			strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// Bindings made by one expression have to be visible to the next, since each is evaluated
// on its own.
func TestStateCarriesAcrossExpressions(t *testing.T) {
	got := runInOrder(t, "ident a = 41;\nprintb a + 1;\n")

	if len(got) == 0 || !strings.Contains(got[len(got)-1], "42") {
		t.Errorf("expected the binding to survive into the next expression, got %v", got)
	}
}
