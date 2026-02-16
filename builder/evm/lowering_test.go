package evm

import (
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

func TestOperandStackDelta(t *testing.T) {
	cases := []struct {
		name string
		op   byte
		want int
	}{
		{name: "OpGetArg_push", op: emitter.OpGetArg, want: 1},
		{name: "OpSave_push", op: emitter.OpSave, want: 1},
		{name: "OpLoad_push", op: emitter.OpLoad, want: 1},
		{name: "OpSubtract_pop2_push1", op: emitter.OpSubtract, want: -1},
		{name: "OpDivide_pop2_push1", op: emitter.OpDivide, want: -1},
		{name: "OpBeginScope_neutral", op: emitter.OpBeginScope, want: 0},
		{name: "OpReturn_neutral", op: emitter.OpReturn, want: 0},
		{name: "OpIdent_neutral", op: emitter.OpIdent, want: 0},
		{name: "OpDefer_neutral", op: emitter.OpDefer, want: 0},
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
		insts []emitter.Instruction
		want  []int
	}{
		{
			name:  "empty",
			insts: nil,
			want:  []int{0},
		},
		{
			name: "single_GetArg",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
			},
			want: []int{0, 1},
		},
		{
			name: "two_GetArg",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
			},
			want: []int{0, 1, 2},
		},
		{
			name: "GetArg_GetArg_Add",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpAdd, []byte("0"), []byte("1")),
			},
			want: []int{0, 1, 2, 2},
		},
		{
			name: "GetArg_GetArg_Sub",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
			},
			want: []int{0, 1, 2, 1},
		},
		{
			name: "GetArg_GetArg_GetArg_Sub_Sub",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpGetArg, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("3"), emitter.OpSubtract, []byte("0"), []byte("1")),
				emitter.NewInstruction([]byte("4"), emitter.OpSubtract, []byte("2"), []byte("3")),
			},
			want: []int{0, 1, 2, 3, 2, 1},
		},
		{
			name: "BeginScope_GetArg_GetArg_Sub_Return",
			insts: []emitter.Instruction{
				emitter.NewInstruction(nil, emitter.OpBeginScope, nil, nil),
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
				emitter.NewInstruction(nil, emitter.OpReturn, nil, nil),
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
		{name: "Sub", op: emitter.OpSubtract, want: true},
		{name: "Div", op: emitter.OpDivide, want: true},
		{name: "Mul", op: emitter.OpMultiply, want: false},
		{name: "Add", op: emitter.OpAdd, want: false},
		{name: "GetArg", op: emitter.OpGetArg, want: false},
		{name: "Return", op: emitter.OpReturn, want: false},
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

func TestReorderLeftAssoc(t *testing.T) {
	cases := []struct {
		name  string
		insts []emitter.Instruction
		want  []emitter.Instruction
	}{
		{
			name:  "empty",
			insts: nil,
			want:  nil,
		},
		{
			name: "single_inst",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
			},
		},
		{
			name: "add_no_reorder",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpAdd, []byte("0"), []byte("1")),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpAdd, []byte("0"), []byte("1")),
			},
		},
		{
			name: "sub_reorder",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
			},
		},
		{
			name: "sub_sub_reorder",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
				emitter.NewInstruction([]byte("3"), emitter.OpGetArg, byteutil.FromUint64(3), nil),
				emitter.NewInstruction([]byte("4"), emitter.OpSubtract, []byte("2"), []byte("3")),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("3"), emitter.OpGetArg, byteutil.FromUint64(3), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
				emitter.NewInstruction([]byte("4"), emitter.OpSubtract, []byte("2"), []byte("3")),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ReorderLeftAssoc(tc.insts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("reorderLeftAssoc(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLowering(t *testing.T) {
	t.Skip()
	cases := []struct {
		name  string
		insts []emitter.Instruction
		want  []emitter.Instruction
	}{
		{
			name: "no_reordering",
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpAdd, []byte("0"), []byte("1")),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpAdd, []byte("0"), []byte("1")),
			},
		},
		{
			name: "reordering",
			// Single Sub: we reorder the instruction sequence (GetArg(1), GetArg(0), Sub) so stack order is correct; IR ops unchanged.
			insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
			},
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("1"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("0"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("2"), emitter.OpSubtract, []byte("0"), []byte("1")),
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
