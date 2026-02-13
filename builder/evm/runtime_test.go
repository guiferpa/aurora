package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

type PickedRuntimeCodeExpectation struct {
	Selector     []byte
	Offset       int
	Length       int
	Instructions []byte
}

func TestPickRuntimeCode_Empty(t *testing.T) {
	cases := []struct {
		Name       string
		SourceCode string
	}{
		{
			"pick_runtime_code_1",
			`{ 4294967295 + 4294967295; };`,
		},
		{
			"pick_runtime_code_2",
			`{ 4294967295 + 4294967295; };
{ true; };`,
		},
		{
			"pick_runtime_code_3",
			`true;
false;
1 + 10_000;`,
		},
	}

	for _, c := range cases {
		bs := bytes.NewBufferString(c.SourceCode).Bytes()
		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: false,
		}).GetFilledTokens(bs)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      "",
			EnableLogging: false,
		}).Parse()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: false,
		}).Emit(ast)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}

		builder := NewBuilder(insts, NewBuilderOptions{EnableLogging: false})
		rc, err := builder.pickRuntimeCode()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		t.Run(c.Name, func(t *testing.T) {
			for _, r := range rc.Referenced {
				got := r.Code.Bytes()

				if len(got) > 0 {
					t.Errorf("EVM do not pick empty runtime: name: %v, got: %v", c.Name, ToString(got))
					return
				} else {
					t.Logf("EVM do no tpick runtime: name: %v, result: %v", c.Name, ToString(got))
				}
			}
		})
	}
}

func TestPickRuntimeCode(t *testing.T) {
	t.SkipNow()
	cases := []struct {
		Name       string
		SourceCode string
		FnExpected func() []PickedRuntimeCodeExpectation
	}{
		{
			"pick_runtime_code_1",
			`ident a = { 4294967295 + 4294967295; };`,
			func() []PickedRuntimeCodeExpectation {
				insts := []byte{OpJumpDestiny, OpPush8}
				insts = append(insts, byteutil.FromUint64(4294967295)...)
				insts = append(insts, OpPush8)
				insts = append(insts, byteutil.FromUint64(4294967295)...)
				insts = append(insts, OpAdd)
				insts = append(insts, OpPush1, 0x00, OpMemoryStore)
				insts = append(insts, OpPush1, 0x20, OpPush1, 0x00, OpReturn)
				insts = append(insts, OpStop)
				expectations := []PickedRuntimeCodeExpectation{
					{
						Selector:     []byte("a"),
						Offset:       0,
						Length:       28,
						Instructions: insts,
					},
				}
				return expectations
			},
		},
		{
			"pick_runtime_code_2",
			`ident a = { 4294967295 + 4294967295; };
ident bcde = { true; };`,
			func() []PickedRuntimeCodeExpectation {
				insts1 := []byte{OpJumpDestiny, OpPush8}
				insts1 = append(insts1, byteutil.FromUint64(4294967295)...)
				insts1 = append(insts1, OpPush8)
				insts1 = append(insts1, byteutil.FromUint64(4294967295)...)
				insts1 = append(insts1, OpAdd)
				insts1 = append(insts1, OpPush1, 0x00, OpMemoryStore)
				insts1 = append(insts1, OpPush1, 0x20, OpPush1, 0x00, OpReturn)
				insts1 = append(insts1, OpStop)

				insts2 := []byte{OpJumpDestiny, OpPush1, 1}
				insts2 = append(insts2, OpPush1, 0x20, OpPush1, 0x00, OpReturn)
				insts2 = append(insts2, OpStop)

				return []PickedRuntimeCodeExpectation{
					{
						Selector:     []byte("a"),
						Offset:       0,
						Length:       28,
						Instructions: insts1,
					},
					{
						Selector:     []byte("bcde"),
						Offset:       29, // 1 (JUMPDEST) + 28 (first block code length)
						Length:       8,
						Instructions: insts2,
					},
				}
			},
		},
	}

	for _, c := range cases {
		bs := bytes.NewBufferString(c.SourceCode).Bytes()
		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: false,
		}).GetFilledTokens(bs)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      "",
			EnableLogging: false,
		}).Parse()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: false,
		}).Emit(ast)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}

		builder := NewBuilder(insts, NewBuilderOptions{EnableLogging: false})
		rc, err := builder.pickRuntimeCode()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		t.Run(c.Name, func(t *testing.T) {
			expecteds := c.FnExpected()
			for i, r := range rc.Referenced {
				got := r.Code.Bytes()

				expected := expecteds[i]

				if !bytes.Equal(got, expected.Instructions) {
					t.Errorf("EVM pick bytecode runtime: name: %v, got: %v, expected: %v", c.Name, ToString(got), ToString(expected.Instructions))
					return
				} else {
					t.Logf("EVM pick runtime: name: %v, result: %v", c.Name, ToString(got))
				}

				if !bytes.Equal(r.Selector, expected.Selector) {
					t.Errorf("EVM pick runtime label: name: %v, got: %s, expected: %s", c.Name, r.Selector, expected.Selector)
					return
				} else {
					t.Logf("EVM pick runtime: name: %v, result: %s", c.Name, r.Selector)
				}

				if expected.Offset != r.Offset {
					t.Errorf("EVM pick runtime offset: name: %v, got: %v, expected: %v", c.Name, r.Offset, expected.Offset)
					return
				}

				if expected.Length != r.Length {
					t.Errorf("EVM pick runtime length: name: %v, got: %v, expected: %v", c.Name, r.Length, expected.Length)
					return
				}
			}
		})
	}
}

func TestBuildRuntimeCode(t *testing.T) {
	t.SkipNow()
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
				expected = append(expected, OpPush1, 0x00, OpMemoryStore)
				expected = append(expected, OpStop)
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
				expected = append(expected, OpPush1, 0x00, OpMemoryStore)
				expected = append(expected, OpStop)
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
				expected = append(expected, OpPush1, 0x00, OpMemoryStore)
				expected = append(expected, OpStop)
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
				expected = append(expected, OpPush1, 0x00, OpMemoryStore)
				expected = append(expected, OpStop)
				return expected
			},
		},
	}

	for _, c := range cases {
		bs := bytes.NewBufferString(c.SourceCode).Bytes()
		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: false,
		}).GetFilledTokens(bs)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      "",
			EnableLogging: false,
		}).Parse()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: false,
		}).Emit(ast)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}

		builder := NewBuilder(insts, NewBuilderOptions{EnableLogging: false})
		bfw, err := builder.buildRuntimeCode()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		got := bfw.Bytes()
		t.Run(c.Name, func(t *testing.T) {
			expected := c.FnExpected()
			if !bytes.Equal(got, expected) {
				t.Errorf("EVM runtime: name: %v, got: %v, expected: %v", c.Name, ToString(got), ToString(expected))
			} else {
				t.Logf("EVM runtime: name: %v, result: %v", c.Name, ToString(got))
			}
		})
	}
}

func TestBuildDispatcher(t *testing.T) {
	cases := []struct {
		Name       string
		FnExpected func() []byte
	}{
		{
			"sample_dispatcher_1",
			func() []byte {
				expected := []byte{OpPush1, 0x00}
				expected = append(expected, OpCallDataLoad)
				expected = append(expected, []byte{OpPush1, 0xe0}...)
				expected = append(expected, OpShiftRight)
				expected = append(expected, []byte{OpPush4, 0x9c, 0x22, 0xff, 0x5f}...)
				expected = append(expected, OpEqual)
				expected = append(expected, []byte{OpPush1, 0x0a}...)
				expected = append(expected, OpJumpIf)
				return expected
			},
		},
	}

	for _, c := range cases {
		bfw, err := buildDispatcher("test", 10)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		got := bfw.Bytes()
		t.Run(c.Name, func(t *testing.T) {
			expected := c.FnExpected()
			if !bytes.Equal(got, expected) {
				t.Errorf("EVM dispatcher: name: %v, got: %v, expected: %v", c.Name, ToString(got), ToString(expected))
			} else {
				t.Logf("EVM dispatcher: name: %v, result: %v", c.Name, ToString(got))
			}
		})

	}
}

func TestGetCalldataArgsOffset(t *testing.T) {
	cases := []struct {
		Name     string
		NthArg   uint64
		Expected byte
	}{
		{
			"sample_get_calldata_args_index_from_bytes_1",
			0,
			0x20, // 32 bytes
		},
		{
			"sample_get_calldata_args_index_from_bytes_2",
			1,
			0x40, // 64 bytes
		},
	}

	for _, c := range cases {
		got := GetCalldataArgsOffset(c.NthArg)
		t.Run(c.Name, func(t *testing.T) {
			expected := c.Expected
			if got != expected {
				t.Errorf("EVM get calldata args index from bytes: name: %v, got: %v, expected: %v", c.Name, got, expected)
			}
		})
	}
}
