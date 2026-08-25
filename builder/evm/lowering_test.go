package evm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// What an instruction takes is read from its operands, not remembered from its opcode. A Ref
// names a value another instruction left behind; an Imm is the value itself and a Const
// belongs to the operation, and neither is waiting on the stack.
func TestConsumesReadsTheOperands(t *testing.T) {
	cases := []struct {
		name string
		inst ir.Instruction
		want int
	}{
		{
			name: "two values",
			inst: ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			want: 2,
		},
		{
			name: "a value and something written down",
			inst: ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.Imm(10, 8)),
			want: 1,
		},
		{
			name: "a value and a number the operation takes",
			inst: ir.NewInstruction([]byte("02"), ir.OpHead, ir.RefTo([]byte("00")), ir.Const(2, 8)),
			want: 1,
		},
		{
			name: "nothing waiting on the stack",
			inst: ir.NewInstruction([]byte("00"), ir.OpSave, ir.Imm(1, 8), ir.Nothing()),
			want: 0,
		},
		{
			name: "as many as a construction has",
			inst: ir.NewInstructionOver([]byte("03"), ir.OpJoin, ir.RefTo([]byte("00")), ir.RefTo([]byte("01")), ir.RefTo([]byte("02"))),
			want: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(consumes(tc.inst)); got != tc.want {
				t.Errorf("takes %d values, want %d", got, tc.want)
			}
		})
	}
}

// The EVM computes "top - next", so a subtraction wants its right operand pushed first. It is
// the machine's rule and not the IR's, which is why it is the one thing here still keyed by
// the opcode.
func TestSubtractionAndDivisionPushTheOtherWayRound(t *testing.T) {
	for _, op := range []byte{ir.OpSubtract, ir.OpDivide} {
		inst := ir.NewInstruction([]byte("02"), op, ir.RefTo([]byte("00")), ir.RefTo([]byte("01")))
		if got := string(consumes(inst)[0].Bytes()); got != "01" {
			t.Errorf("%s pushes %q first, want the right operand", ir.ResolveOpCode(op), got)
		}
	}
	inst := ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("01")))
	if got := string(consumes(inst)[0].Bytes()); got != "00" {
		t.Errorf("an addition pushes %q first, want the left operand", got)
	}
}

func TestResolveOperandsOrderFromSourceCode(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   []ir.Instruction
	}{
		{
			name:   "single_inst",
			source: "feed(0);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
			},
		},
		{
			name:   "add_no_reorder",
			source: "feed(0) + feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			name:   "sub_reorder",
			source: "feed(0) - feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			name:   "sub_sub_reorder",
			source: "feed(0) - feed(1) - feed(2) - feed(3);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpGetFeed, ir.Const(3, 0), ir.Nothing()),
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, ir.Const(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, ir.RefTo([]byte("02")), ir.RefTo([]byte("03"))),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, ir.RefTo([]byte("04")), ir.RefTo([]byte("05"))),
			},
		},
		{
			// A chain of divisions has the same shape as a chain of subtractions — both
			// group to the left and neither is associative — so the lowering has to
			// reorder it the same way. Nothing covered this before: the tree used to
			// group division to the right, where the question did not arise.
			name:   "div_div_reorder",
			source: "feed(0) / feed(1) / feed(2);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, ir.Const(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpDivide, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
				ir.NewInstruction([]byte("04"), ir.OpDivide, ir.RefTo([]byte("02")), ir.RefTo([]byte("03"))),
			},
		},
		{
			name:   "div_reorder",
			source: "feed(0) / feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpDivide, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			// Literals are written into the instructions that use them, so what is left is
			// the shape of the arithmetic and nothing else. Division is left-associative and
			// binds tighter, so "1 - 2 / 2 - 1" is "(1 - (2 / 2)) - 1".
			name:   "div_and_sub_reorder",
			source: "1 - 2 / 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpDivide, ir.Imm(2, 0), ir.Imm(2, 0)),
				ir.NewInstruction([]byte("01"), ir.OpSubtract, ir.Imm(1, 0), ir.RefTo([]byte("00"))),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, ir.RefTo([]byte("01")), ir.Imm(1, 0)),
			},
		},
		{
			name:   "sub_and_mult_reorder",
			source: "6 - 2 * 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpMultiply, ir.Imm(2, 0), ir.Imm(2, 0)),
				ir.NewInstruction([]byte("01"), ir.OpSubtract, ir.Imm(6, 0), ir.RefTo([]byte("00"))),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, ir.RefTo([]byte("01")), ir.Imm(1, 0)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBufferString(tc.source).Bytes()
			tokens, err := lexer.New().GetFilledTokens(bs)
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			tree, err := parser.New().Parse(parser.ParseInput{Tokens: tokens})
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			// Compared as the IR reads rather than as the struct is: this is about the
			// order instructions come out in, and an instruction also carries where it was
			// written, which the expectations below have no business restating.
			got := ResolveOperandsOrder(insts[0].Insts, 0)
			if ir.Format(got) != ir.Format(tc.want) {
				t.Errorf("\ngot =\n%v \nwant =\n%v", ir.Format(got), ir.Format(tc.want))
			}
		})
	}
}

func TestLowering(t *testing.T) {
	cases := []struct {
		name  string
		insts []ir.Instruction
		want  []ir.Instruction
	}{
		{
			name: "no_reordering",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpAdd, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpAdd, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
		},
		{
			name: "reordering",
			// Single Sub: we reorder the instruction sequence (GetArg(1), GetArg(0), Sub) so stack order is correct; IR ops unchanged.
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.Const(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.Const(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Lowering(tc.insts, 0)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Lowering(%v) = %v, want %v", tc.insts, got, tc.want)
			}
		})
	}
}

// A value is held back until whoever takes it, and no further: past a branch there is no
// telling whether it would have run. A push moved into an arm happens only when that arm is
// taken, and whoever eats it may be reached the other way — so it goes out before the branch,
// where it is still on the straight run it was written on.
func TestNothingIsMovedAcrossABranch(t *testing.T) {
	insts := []ir.Instruction{
		// A value written before the branch, and read inside it.
		ir.NewInstruction([]byte("00"), ir.OpSave, ir.Imm(1, 0), ir.Nothing()),
		ir.NewInstruction([]byte("01"), ir.OpSave, ir.Imm(2, 0), ir.Nothing()),
		ir.NewInstruction([]byte("02"), ir.OpIf, ir.RefTo([]byte("01")), ir.TargetAt(2)),
		ir.NewInstruction([]byte("03"), ir.OpSave, ir.Imm(3, 0), ir.Nothing()),
		ir.NewInstruction([]byte("04"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("03"))),
	}

	lowered := ResolveOperandsOrder(insts, 0)

	at := func(label string) int {
		for i, inst := range lowered {
			if string(inst.GetLabel()) == label {
				return i
			}
		}
		t.Fatalf("%s is not in the lowered scope:\n%s", label, ir.Format(lowered))
		return -1
	}

	if at("00") > at("02") {
		t.Errorf("the value written before the branch was moved past it:\n%s", ir.Format(lowered))
	}
	// And the test of the branch is still put right in front of it.
	if at("01") != at("02")-1 {
		t.Errorf("what the branch tests is not in front of it:\n%s", ir.Format(lowered))
	}
}
