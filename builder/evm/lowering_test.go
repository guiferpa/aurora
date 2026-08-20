package evm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// The depth after each instruction, which is what says a scope is balanced — and what the two
// sides of a branch will have to agree on when there are branches.
func TestStackDepth(t *testing.T) {
	cases := []struct {
		name  string
		insts []ir.Instruction
		want  []int
	}{
		{
			name:  "empty",
			insts: nil,
			want:  []int{0},
		},
		{
			name: "one argument is one value",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
			},
			want: []int{0, 1},
		},
		{
			name: "two arguments and a sum leave one",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("2"), ir.OpAdd, []byte("0"), []byte("1")),
			},
			want: []int{0, 1, 2, 1},
		},
		{
			name: "a binding takes its value and leaves nothing",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpIdent, []byte("side"), []byte("0")),
			},
			want: []int{0, 1, 0},
		},
		{
			name: "a return takes the answer with it",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpReturn, nil, []byte("0")),
			},
			want: []int{0, 1, 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StackDepth(tc.insts); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("StackDepth(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

// What an instruction takes, and in which order it has to reach the stack. Subtraction and
// division read theirs the other way round, which is the one case worth a table.
func TestWhatAnInstructionTakes(t *testing.T) {
	cases := []struct {
		name string
		op   byte
		want []field
	}{
		{name: "a sum takes both, left first", op: ir.OpAdd, want: []field{fieldLeft, fieldRight}},
		{name: "a subtraction takes them the other way round", op: ir.OpSubtract, want: []field{fieldRight, fieldLeft}},
		{name: "a division too", op: ir.OpDivide, want: []field{fieldRight, fieldLeft}},
		{name: "a binding takes the value it binds", op: ir.OpIdent, want: []field{fieldRight}},
		{name: "a return takes what it answers with", op: ir.OpReturn, want: []field{fieldRight}},
		{name: "an argument takes nothing", op: ir.OpGetFeed, want: nil},
		{name: "opening a scope takes nothing", op: ir.OpBeginScope, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := consumes(tc.op); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("consumes(0x%02x) = %v, want %v", tc.op, got, tc.want)
			}
		})
	}
}

// What is left on the stack afterwards. A binding is the one that surprises: it writes to
// memory and the stack comes out as it went in.
func TestWhatAnInstructionLeaves(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   byte
		want bool
	}{
		{name: "a constant", op: ir.OpSave, want: true},
		{name: "an argument", op: ir.OpGetFeed, want: true},
		{name: "a read of a name", op: ir.OpLoad, want: true},
		{name: "a sum", op: ir.OpAdd, want: true},
		{name: "a binding", op: ir.OpIdent, want: false},
		{name: "a return", op: ir.OpReturn, want: false},
		{name: "opening a scope", op: ir.OpBeginScope, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := produces(tc.op); got != tc.want {
				t.Errorf("produces(0x%02x) = %v, want %v", tc.op, got, tc.want)
			}
		})
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
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
			},
		},
		{
			name:   "add_no_reorder",
			source: "feed(0) + feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("02"), ir.OpAdd, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "sub_reorder",
			source: "feed(0) - feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "sub_sub_reorder",
			source: "feed(0) - feed(1) - feed(2) - feed(3);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpGetFeed, byteutil.FromUint64(3), nil),
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, []byte("00"), []byte("01")),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, []byte("02"), []byte("03")),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, []byte("04"), []byte("05")),
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
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("02"), ir.OpDivide, []byte("00"), []byte("01")),
				ir.NewInstruction([]byte("04"), ir.OpDivide, []byte("02"), []byte("03")),
			},
		},
		{
			name:   "div_reorder",
			source: "feed(0) / feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("02"), ir.OpDivide, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "div_and_sub_reorder",
			source: "1 - 2 / 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpSave, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("02"), ir.OpSave, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("01"), ir.OpSave, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("03"), ir.OpDivide, []byte("01"), []byte("02")),
				ir.NewInstruction([]byte("00"), ir.OpSave, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, []byte("00"), []byte("03")),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, []byte("04"), []byte("05")),
			},
		},
		{
			name:   "sub_and_mult_reorder",
			source: "6 - 2 * 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpSave, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("01"), ir.OpSave, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("02"), ir.OpSave, byteutil.FromUint64(2), nil),
				ir.NewInstruction([]byte("03"), ir.OpMultiply, []byte("01"), []byte("02")),
				ir.NewInstruction([]byte("00"), ir.OpSave, byteutil.FromUint64(6), nil),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, []byte("00"), []byte("03")),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, []byte("04"), []byte("05")),
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
			got := ResolveOperandsOrder(insts, 0)
			if !reflect.DeepEqual(got, tc.want) {
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
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("2"), ir.OpAdd, []byte("0"), []byte("1")),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("2"), ir.OpAdd, []byte("0"), []byte("1")),
			},
		},
		{
			name: "reordering",
			// Single Sub: we reorder the instruction sequence (GetArg(1), GetArg(0), Sub) so stack order is correct; IR ops unchanged.
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, []byte("0"), []byte("1")),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, byteutil.FromUint64(1), nil),
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, byteutil.FromUint64(0), nil),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, []byte("0"), []byte("1")),
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
