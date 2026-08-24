package emitter

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

func compileWith(t *testing.T, source string, tapeSize int) ir.Program {
	t.Helper()
	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens, TapeSize: tapeSize})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	program, err := New(NewEmitterOptions{TapeSize: tapeSize}).EmitProgram(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}
	return program
}

func compile(t *testing.T, source string) ir.Program {
	t.Helper()
	return compileWith(t, source, 0)
}

// A program reports one expression per top-level node, in source order, and the ranges
// tile the instruction stream without gaps.
func TestEmitProgramCoversEveryTopLevelExpression(t *testing.T) {
	program := compile(t, "ident a = 1;\nprintb a;\n2 + 3;\n")

	if len(program.Expressions) != 3 {
		t.Fatalf("expected 3 expressions, got %d", len(program.Expressions))
	}

	previous := 0
	for i, expr := range program.Expressions {
		if expr.From != previous {
			t.Errorf("expression %d starts at %d, want %d", i, expr.From, previous)
		}
		if expr.To <= expr.From {
			t.Errorf("expression %d is empty: %+v", i, expr)
		}
		if len(expr.Label) == 0 {
			t.Errorf("expression %d has no label", i)
		}
		previous = expr.To
	}
	if previous != len(program.Instructions) {
		t.Errorf("expressions cover %d instructions, the program has %d", previous, len(program.Instructions))
	}
}

// Emit keeps returning just the stream, so nothing that only wants instructions changes.
func TestEmitMatchesEmitProgram(t *testing.T) {
	const source = "ident a = 1;\nprintb a + 1;\n"
	program := compile(t, source)

	tokens, _ := lexer.New().GetFilledTokens([]byte(source))
	tree, _ := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
	insts, err := New(NewEmitterOptions{}).Emit(tree)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if len(insts) != len(program.Instructions) {
		t.Fatalf("Emit gave %d instructions, EmitProgram %d", len(insts), len(program.Instructions))
	}
	for i := range insts {
		if !bytes.Equal(insts[i].GetLabel(), program.Instructions[i].GetLabel()) {
			t.Errorf("instruction %d differs", i)
		}
	}
}

// The label of a scope is where its ir.OpReturn writes, which is what makes the value of a
// block or an if readable at all.
func TestEmitProgramLabelsScopes(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "block", source: "{ 7; };"},
		{name: "if", source: "if true { 7; };"},
		{name: "if else", source: "if false { 1; } else { 7; };"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := compile(t, tc.source)
			if len(program.Expressions) != 1 {
				t.Fatalf("expected one expression, got %d", len(program.Expressions))
			}
			expr := program.Expressions[0]

			last := program.Instructions[expr.To-1]
			if last.GetOpCode() != ir.OpReturn {
				t.Fatalf("a scope should end in OpReturn, got %s", ir.ResolveOpCode(last.GetOpCode()))
			}
			if !bytes.Equal(last.GetLeft().Bytes(), expr.Label) {
				t.Errorf("the value lands under %q but the expression reports %q", last.GetLeft().Bytes(), expr.Label)
			}
		})
	}
}

// A field says how many tapes the run it reads has, beside the index it reads at.
//
// A run is tapes laid end to end with nothing in it saying where it ends, so where a field
// sits cannot be worked out from the index alone: reading the last tape of a run means
// counting from the end. Whoever reads the IR cannot work it out either — the run may arrive
// under a name, as a value applied to a scope, or as a field of another run, and in none of
// those is the construction in sight.
func TestAFieldSaysHowLongTheRunIs(t *testing.T) {
	const source = `shape Point { x, y };
shape Line { a, b, c };
ident p = Point{1, 2};
ident l = Line{1, 2, 3};
printd p.y;
printd l.c;`

	wanted := []uint64{2, 3}
	read := make([]uint64, 0, len(wanted))
	for _, inst := range compile(t, source).Instructions {
		if inst.GetOpCode() != ir.OpField {
			continue
		}
		operands := inst.GetOperands()
		if len(operands) != 3 {
			t.Fatalf("a field carries %d operands, want the run it reads, the index, and the length", len(operands))
		}
		read = append(read, byteutil.ToUint64(operands[2].Bytes()))
	}

	if len(read) != len(wanted) {
		t.Fatalf("found %d fields, want %d", len(read), len(wanted))
	}
	for at, want := range wanted {
		if read[at] != want {
			t.Errorf("the field reads a run of %d tapes, want %d", read[at], want)
		}
	}
}
