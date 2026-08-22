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

func TestOperandStackDelta(t *testing.T) {
	cases := []struct {
		name string
		op   byte
		want int
	}{
		{name: "OpGetFeed_push", op: ir.OpGetFeed, want: 1},
		{name: "OpSave_push", op: ir.OpSave, want: 1},
		{name: "OpLoad_push", op: ir.OpLoad, want: 1},
		{name: "OpSubtract_pop2_push1", op: ir.OpSubtract, want: -1},
		{name: "OpDivide_pop2_push1", op: ir.OpDivide, want: -1},
		{name: "OpBeginScope_neutral", op: ir.OpBeginScope, want: 0},
		{name: "OpReturn_neutral", op: ir.OpReturn, want: 0},
		{name: "OpIdent_neutral", op: ir.OpIdent, want: 0},
		{name: "OpDefer_neutral", op: ir.OpDefer, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := OperandStackDelta(tc.op)
			if got != tc.want {
				t.Errorf("OperandStackDelta(0x%02x) = %d, want %d", tc.op, got, tc.want)
			}
		})
	}
}

func TestGetOperandStackDeltaDepth(t *testing.T) {
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
			name: "single_GetArg",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
			},
			want: []int{0, 1},
		},
		{
			name: "two_GetArg",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
			},
			want: []int{0, 1, 2},
		},
		{
			name: "GetArg_GetArg_Add",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpAdd, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []int{0, 1, 2, 2},
		},
		{
			name: "GetArg_GetArg_Sub",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []int{0, 1, 2, 1},
		},
		{
			name: "GetArg_GetArg_GetArg_Sub_Sub",
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpGetFeed, ir.ConstNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("3"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
				ir.NewInstruction([]byte("4"), ir.OpSubtract, ir.RefTo([]byte("2")), ir.RefTo([]byte("3"))),
			},
			want: []int{0, 1, 2, 3, 2, 1},
		},
		{
			name: "BeginScope_GetArg_GetArg_Sub_Return",
			insts: []ir.Instruction{
				ir.NewInstruction(nil, ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
				ir.NewInstruction(nil, ir.OpReturn, ir.RefTo(nil), ir.RefTo(nil)),
			},
			want: []int{0, 0, 1, 2, 1, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetOperandStackDeltaDepth(tc.insts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("GetOperandStackDeltaDepth(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsAsociativeOperator(t *testing.T) {
	cases := []struct {
		name string
		op   byte
		want bool
	}{
		{name: "Sub", op: ir.OpSubtract, want: true},
		{name: "Div", op: ir.OpDivide, want: true},
		{name: "Mul", op: ir.OpMultiply, want: false},
		{name: "Add", op: ir.OpAdd, want: false},
		{name: "GetArg", op: ir.OpGetFeed, want: false},
		{name: "Return", op: ir.OpReturn, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAssociativeOperator(tc.op)
			if got != tc.want {
				t.Errorf("IsAssociativeOperator(0x%02x) = %v, want %v", tc.op, got, tc.want)
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
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
			},
		},
		{
			name:   "add_no_reorder",
			source: "feed(0) + feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			name:   "sub_reorder",
			source: "feed(0) - feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpSubtract, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			name:   "sub_sub_reorder",
			source: "feed(0) - feed(1) - feed(2) - feed(3);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpGetFeed, ir.ConstNum(3, 0), ir.Nothing()),
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, ir.ConstNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
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
				ir.NewInstruction([]byte("03"), ir.OpGetFeed, ir.ConstNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpDivide, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
				ir.NewInstruction([]byte("04"), ir.OpDivide, ir.RefTo([]byte("02")), ir.RefTo([]byte("03"))),
			},
		},
		{
			name:   "div_reorder",
			source: "feed(0) / feed(1);",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("01"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpDivide, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
			},
		},
		{
			name:   "div_and_sub_reorder",
			source: "1 - 2 / 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpSave, ir.ImmNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpSave, ir.ImmNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpSave, ir.ImmNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("03"), ir.OpDivide, ir.RefTo([]byte("01")), ir.RefTo([]byte("02"))),
				ir.NewInstruction([]byte("00"), ir.OpSave, ir.ImmNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, ir.RefTo([]byte("00")), ir.RefTo([]byte("03"))),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, ir.RefTo([]byte("04")), ir.RefTo([]byte("05"))),
			},
		},
		{
			name:   "sub_and_mult_reorder",
			source: "6 - 2 * 2 - 1;",
			want: []ir.Instruction{
				ir.NewInstruction([]byte("05"), ir.OpSave, ir.ImmNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("01"), ir.OpSave, ir.ImmNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("02"), ir.OpSave, ir.ImmNum(2, 0), ir.Nothing()),
				ir.NewInstruction([]byte("03"), ir.OpMultiply, ir.RefTo([]byte("01")), ir.RefTo([]byte("02"))),
				ir.NewInstruction([]byte("00"), ir.OpSave, ir.ImmNum(6, 0), ir.Nothing()),
				ir.NewInstruction([]byte("04"), ir.OpSubtract, ir.RefTo([]byte("00")), ir.RefTo([]byte("03"))),
				ir.NewInstruction([]byte("06"), ir.OpSubtract, ir.RefTo([]byte("04")), ir.RefTo([]byte("05"))),
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
			got := ResolveOperandsOrder(insts)
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
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpAdd, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpAdd, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
		},
		{
			name: "reordering",
			// Single Sub: we reorder the instruction sequence (GetArg(1), GetArg(0), Sub) so stack order is correct; IR ops unchanged.
			insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
			want: []ir.Instruction{
				ir.NewInstruction([]byte("1"), ir.OpGetFeed, ir.ConstNum(1, 0), ir.Nothing()),
				ir.NewInstruction([]byte("0"), ir.OpGetFeed, ir.ConstNum(0, 0), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpSubtract, ir.RefTo([]byte("0")), ir.RefTo([]byte("1"))),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Lowering(tc.insts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Lowering(%v) = %v, want %v", tc.insts, got, tc.want)
			}
		})
	}
}
