package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func TestBuildRuntimeCode(t *testing.T) {
	cases := []struct {
		Name       string
		SourceCode string
		FnExpected func() []byte
	}{
		{
			"math_sum_1",
			`4294967295 + 4294967295;`,
			func() []byte {
				expected := []byte{OpPush8}
				expected = append(expected, byteutil.FromUint64(4294967295)...)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(4294967295)...)
				expected = append(expected, OpAdd)
				return expected
			},
		},
		{
			"math_sum_and_multiply_2",
			`3 + 3 * 2;`,
			func() []byte {
				expected := []byte{OpPush8}
				expected = append(expected, byteutil.FromUint64(2)...)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(3)...)
				expected = append(expected, OpMul)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(3)...)
				expected = append(expected, OpAdd)
				return expected
			},
		},
		{
			"math_sub_and_multiply_2",
			`10 - 3 * 2;`,
			func() []byte {
				expected := []byte{OpPush8}
				expected = append(expected, byteutil.FromUint64(2)...)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(3)...)
				expected = append(expected, OpMul)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(10)...)
				expected = append(expected, OpSub)
				return expected
			},
		},
		{
			"math_sub_and_divide_2",
			`3 - 2 / 2;`,

			func() []byte {
				expected := []byte{OpPush8}
				expected = append(expected, byteutil.FromUint64(2)...)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(2)...)
				expected = append(expected, OpDiv)
				expected = append(expected, OpPush8)
				expected = append(expected, byteutil.FromUint64(3)...)
				expected = append(expected, OpSub)
				return expected
			},
		},
	}

	builder := NewBuilder()

	for _, c := range cases {
		bs := bytes.NewBufferString(c.SourceCode).Bytes()
		tokens, err := lexer.GetFilledTokens(bs)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		insts, err := emitter.New().Emit(ast)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		bfr, err := builder.buildRuntimeCode(insts)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		got := bfr.Bytes()
		t.Run(c.Name, func(t *testing.T) {
			expected := c.FnExpected()
			if !bytes.Equal(got, expected) {
				t.Errorf("EVM transformer: name: %v, got: %v, expected: %v", c.Name, ToString(got), ToString(expected))
			} else {
				t.Logf("EVM transformer: name: %v, result: %v", c.Name, ToString(got))
			}
		})
	}
}

func TestBuildInitCode(t *testing.T) {
	builder := NewBuilder()
	bfr, err := builder.buildInitCode(5)
	if err != nil {
		t.Errorf("Error building init code: %v", err)
		return
	}
	got := bfr.Bytes()
	expected := []byte{OpPush1, 5, OpPush1, 0x0c, OpPush1, 0x00, OpCodeCopy, OpPush1, 5, OpPush1, 0x00, OpReturn}
	if !bytes.Equal(got, expected) {
		t.Errorf("Init code: got: %v, expected: %v", ToString(got), ToString(expected))
	}
}
