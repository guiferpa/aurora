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

// everyInstruction answers every instruction of a program, whichever block it landed in. A
// test about what an instruction carries has no business knowing which block that is.
func everyInstruction(blocks []ir.Block) []ir.Instruction {
	insts := make([]ir.Instruction, 0)
	for _, block := range blocks {
		insts = append(insts, block.Insts...)
	}
	return insts
}

// A program reports one expression per top-level node, in source order, and each one says
// where it begins among the blocks — which is what lets a runner answer where each expression
// happens rather than all of them at the end.
//
// They begin one after another, which is the whole of what "in order" means when the thing
// they are places in is no longer a list: a later expression is either further into the same
// block or in a block reached after it.
func TestEmitProgramCoversEveryTopLevelExpression(t *testing.T) {
	program := compile(t, "ident a = 1;\nprintb a;\n2 + 3;\n")

	if len(program.Expressions) != 3 {
		t.Fatalf("expected 3 expressions, got %d", len(program.Expressions))
	}

	previous := ir.Point{}
	for i, expr := range program.Expressions {
		if expr.At.Block < previous.Block || (expr.At.Block == previous.Block && expr.At.At < previous.At) {
			t.Errorf("expression %d begins at b%d@%d, which is before b%d@%d",
				i, expr.At.Block, expr.At.At, previous.Block, previous.At)
		}
		if i > 0 && expr.At == previous {
			t.Errorf("expression %d begins where %d does, and they are two", i, i-1)
		}
		if len(expr.Label) == 0 {
			t.Errorf("expression %d has no label", i)
		}
		previous = expr.At
	}
	if int(previous.Block) >= len(program.Blocks) {
		t.Errorf("the last expression is in block %d, and the program has %d", previous.Block, len(program.Blocks))
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

	if len(insts) != len(program.Blocks) {
		t.Fatalf("Emit gave %d blocks, EmitProgram %d", len(insts), len(program.Blocks))
	}
	for i := range insts {
		if ir.FormatBlocks(insts[i:i+1]) != ir.FormatBlocks(program.Blocks[i:i+1]) {
			t.Errorf("block %d differs", i)
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

			// The value of a scope reaches whoever reads it by being handed over: the block
			// the run carries on into takes it, under the name the expression returns under.
			taken := false
			for _, block := range program.Blocks {
				for _, param := range block.Params {
					if bytes.Equal(param, expr.Label) {
						taken = true
					}
				}
			}
			if !taken {
				t.Errorf("no block takes %q, and that is the name the expression returns under", expr.Label)
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
	for _, inst := range everyInstruction(compile(t, source).Blocks) {
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
