package evm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
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

func TestResolveOperandsOrderFromSourceCode(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   []emitter.Instruction
	}{
		{
			name:   "single_inst",
			source: "arguments(0);",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("00"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
			},
		},
		{
			name:   "add_no_reorder",
			source: "arguments(0) + arguments(1);",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("00"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("01"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpAdd, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "sub_reorder",
			source: "arguments(0) - arguments(1);",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("01"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("00"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpSubtract, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "sub_sub_reorder",
			source: "arguments(0) - arguments(1) - arguments(2) - arguments(3);",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("05"), emitter.OpGetArg, byteutil.FromUint64(3), nil),
				emitter.NewInstruction([]byte("03"), emitter.OpGetArg, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("01"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("00"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpSubtract, []byte("00"), []byte("01")),
				emitter.NewInstruction([]byte("04"), emitter.OpSubtract, []byte("02"), []byte("03")),
				emitter.NewInstruction([]byte("06"), emitter.OpSubtract, []byte("04"), []byte("05")),
			},
		},
		{
			name:   "div_reorder",
			source: "arguments(0) / arguments(1);",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("01"), emitter.OpGetArg, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("00"), emitter.OpGetArg, byteutil.FromUint64(0), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpDivide, []byte("00"), []byte("01")),
			},
		},
		{
			name:   "div_and_sub_reorder",
			source: "1 - 2 / 2 - 1;",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("05"), emitter.OpSave, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpSave, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("01"), emitter.OpSave, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("03"), emitter.OpDivide, []byte("01"), []byte("02")),
				emitter.NewInstruction([]byte("00"), emitter.OpSave, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("04"), emitter.OpSubtract, []byte("00"), []byte("03")),
				emitter.NewInstruction([]byte("06"), emitter.OpSubtract, []byte("04"), []byte("05")),
			},
		},
		{
			name:   "sub_and_mult_reorder",
			source: "6 - 2 * 2 - 1;",
			want: []emitter.Instruction{
				emitter.NewInstruction([]byte("05"), emitter.OpSave, byteutil.FromUint64(1), nil),
				emitter.NewInstruction([]byte("01"), emitter.OpSave, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("02"), emitter.OpSave, byteutil.FromUint64(2), nil),
				emitter.NewInstruction([]byte("03"), emitter.OpMultiply, []byte("01"), []byte("02")),
				emitter.NewInstruction([]byte("00"), emitter.OpSave, byteutil.FromUint64(6), nil),
				emitter.NewInstruction([]byte("04"), emitter.OpSubtract, []byte("00"), []byte("03")),
				emitter.NewInstruction([]byte("06"), emitter.OpSubtract, []byte("04"), []byte("05")),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBufferString(tc.source).Bytes()
			tokens, err := lexer.New(lexer.NewLexerOptions{
				EnableLogging: false,
			}).GetFilledTokens(bs)
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			ast, err := parser.New(tokens, parser.NewParserOptions{
				EnableLogging: false,
			}).Parse()
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{
				EnableLogging: false,
			}).Emit(ast)
			if err != nil {
				t.Errorf("%v: %v", tc.name, err)
				return
			}
			got := ResolveOperandsOrder(insts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("\ngot =\n%v \nwant =\n%v", emitter.Format(got), emitter.Format(tc.want))
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
