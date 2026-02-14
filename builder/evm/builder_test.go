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
		rc, err := builder.PickRuntimeCode()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		t.Run(c.Name, func(t *testing.T) {
			for _, r := range rc.Dispatchers {
				got := r.Code.Bytes()

				if len(got) > 0 {
					t.Errorf("EVM do not pick empty runtime: name: %v, got: %v", c.Name, byteutil.ToUpperHex(got))
					return
				} else {
					t.Logf("EVM do not pick runtime: name: %v, result: %v", c.Name, byteutil.ToUpperHex(got))
				}
			}
		})
	}
}

func TestPickRuntimeCode(t *testing.T) {
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
				insts2 = append(insts2, OpPush1, 0x00, OpMemoryStore)
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
						Length:       11, // JUMPDEST + PUSH1 1 + PUSH1 0 MSTORE + RETURN (32 bytes from 0)
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
		rc, err := builder.PickRuntimeCode()
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		t.Run(c.Name, func(t *testing.T) {
			expecteds := c.FnExpected()
			for i, d := range rc.Dispatchers {
				got := d.Code.Bytes()

				expected := expecteds[i]

				if !bytes.Equal(got, expected.Instructions) {
					t.Errorf("EVM pick bytecode runtime: name: %v, got: %v, expected: %v", c.Name, byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected.Instructions))
					return
				} else {
					t.Logf("EVM pick runtime: name: %v, result: %v", c.Name, byteutil.ToUpperHex(got))
				}

				if !bytes.Equal(d.Selector, expected.Selector) {
					t.Errorf("EVM pick runtime label: name: %v, got: %s, expected: %s", c.Name, d.Selector, expected.Selector)
					return
				} else {
					t.Logf("EVM pick runtime: name: %v, result: %s", c.Name, d.Selector)
				}

				if expected.Offset != d.Offset {
					t.Errorf("EVM pick runtime offset: name: %v, got: %v, expected: %v", c.Name, d.Offset, expected.Offset)
					return
				}

				if expected.Length != d.Length {
					t.Errorf("EVM pick runtime length: name: %v, got: %v, expected: %v", c.Name, d.Length, expected.Length)
					return
				}
			}
		})
	}
}

func TestPickDeferAtCursor(t *testing.T) {
	cases := []struct {
		Name                 string
		Insts                []emitter.Instruction
		Cursor               int
		Offset               int
		WantOK               bool
		WantNextCursor       int
		WantSelector         string // only checked when WantOK
		WantDispatcherOffset int    // only checked when WantOK
		WantCodeNonEmpty     bool   // only checked when WantOK
	}{
		{
			Name: "valid_defer",
			Insts: []emitter.Instruction{
				emitter.NewInstruction([]byte("0"), emitter.OpDefer, []byte("ret"), byteutil.FromUint64(2)),
				emitter.NewInstruction([]byte("1"), emitter.OpBeginScope, nil, nil),
				emitter.NewInstruction([]byte("2"), emitter.OpReturn, nil, nil),
				emitter.NewInstruction([]byte("3"), emitter.OpIdent, []byte("f"), []byte("0")),
			},
			Cursor:               0,
			Offset:               0,
			WantOK:               true,
			WantNextCursor:       3,
			WantSelector:         "f",
			WantDispatcherOffset: 0,
			WantCodeNonEmpty:     true,
		},
		{
			Name: "not_op_defer",
			Insts: []emitter.Instruction{
				emitter.NewInstruction(nil, emitter.OpBeginScope, nil, nil),
			},
			Cursor:         0,
			Offset:         0,
			WantOK:         false,
			WantNextCursor: 0,
		},
		{
			Name: "defer_without_op_ident_after",
			Insts: []emitter.Instruction{
				emitter.NewInstruction(nil, emitter.OpDefer, nil, byteutil.FromUint64(2)),
				emitter.NewInstruction(nil, emitter.OpBeginScope, nil, nil),
				emitter.NewInstruction(nil, emitter.OpReturn, nil, nil),
				emitter.NewInstruction(nil, emitter.OpAdd, nil, nil),
			},
			Cursor:         0,
			Offset:         0,
			WantOK:         false,
			WantNextCursor: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			b := NewBuilder(c.Insts, NewBuilderOptions{EnableLogging: false})
			d, nextCursor, ok := b.PickDeferAtCursor(c.Cursor, c.Offset)
			if ok != c.WantOK {
				t.Errorf("ok = %v, want %v", ok, c.WantOK)
			}
			if nextCursor != c.WantNextCursor {
				t.Errorf("nextCursor = %d, want %d", nextCursor, c.WantNextCursor)
			}
			if !c.WantOK {
				return
			}
			if d == nil {
				t.Fatal("dispatcher is nil")
			}
			if c.WantSelector != "" && string(d.Selector) != c.WantSelector {
				t.Errorf("selector = %q, want %q", d.Selector, c.WantSelector)
			}
			if d.Offset != c.WantDispatcherOffset {
				t.Errorf("Offset = %d, want %d", d.Offset, c.WantDispatcherOffset)
			}
			if c.WantCodeNonEmpty && (d.Code == nil || d.Code.Len() == 0) {
				t.Error("expected dispatcher code to be non-empty")
			}
		})
	}
}

func TestNewIdentManager(t *testing.T) {
	m := NewIdentManager()
	if m == nil {
		t.Fatal("NewIdentManager returned nil")
	}
	if n := m.GetLength(); n != 0 {
		t.Errorf("new IdentManager should have length 0, got %d", n)
	}
}

func TestIdentManager_GetOffset(t *testing.T) {
	m := NewIdentManager()
	got := m.GetOffset([]byte("x"))
	if got != 0 {
		t.Errorf("GetOffset for missing ident should return 0, got %d", got)
	}
}

func TestIdentManager_SetOffset_GetOffset(t *testing.T) {
	m := NewIdentManager()
	m.SetOffset("a", 0)
	m.SetOffset("b", 32)

	if got := m.GetOffset([]byte("a")); got != 0 {
		t.Errorf("GetOffset(a) = %d, want 0", got)
	}
	if got := m.GetOffset([]byte("b")); got != 32 {
		t.Errorf("GetOffset(b) = %d, want 32", got)
	}
	if got := m.GetOffset([]byte("c")); got != 0 {
		t.Errorf("GetOffset(c) for missing ident = %d, want 0", got)
	}
}

func TestIdentManager_GetLength(t *testing.T) {
	m := NewIdentManager()
	if n := m.GetLength(); n != 0 {
		t.Errorf("initial GetLength = %d, want 0", n)
	}
	m.SetOffset("a", 0)
	if n := m.GetLength(); n != 1 {
		t.Errorf("after one SetOffset, GetLength = %d, want 1", n)
	}
	m.SetOffset("b", 32)
	if n := m.GetLength(); n != 2 {
		t.Errorf("after two SetOffset, GetLength = %d, want 2", n)
	}
}

func TestIdentManager_SetOffset_overwrites(t *testing.T) {
	m := NewIdentManager()
	m.SetOffset("x", 0)
	m.SetOffset("x", 64)
	if got := m.GetOffset([]byte("x")); got != 64 {
		t.Errorf("GetOffset(x) after overwrite = %d, want 64", got)
	}
	if n := m.GetLength(); n != 1 {
		t.Errorf("overwriting same key should keep length 1, got %d", n)
	}
}
