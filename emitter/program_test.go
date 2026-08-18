package emitter

import (
	"bytes"
	"testing"

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
			if !bytes.Equal(last.GetLeft(), expr.Label) {
				t.Errorf("the value lands under %q but the expression reports %q", last.GetLeft(), expr.Label)
			}
		})
	}
}
