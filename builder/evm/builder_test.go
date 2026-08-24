package evm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
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
		tokens, err := lexer.New().GetFilledTokens(bs)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		tree, err := parser.New().Parse(parser.ParseInput{Tokens: tokens})
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}

		builder := NewBuilder(insts, NewBuilderOptions{})
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
		FnExpected func(got []byte) error
	}{
		{
			"callable_scope_with_add",
			`ident a = { 4294967295 + 4294967295; };`,
			//nolint:errcheck
			func(got []byte) error {
				want := bytes.NewBuffer(make([]byte, 0))
				WritePush(want, byteutil.FromUint64(4294967295), byteutil.DefaultTapeSize)
				WritePush(want, byteutil.FromUint64(4294967295), byteutil.DefaultTapeSize)
				WriteAdd(want)
				WriteMask(want, byteutil.DefaultTapeSize)
				WriteAnswer(want)
				// The scope reads no position, so its names begin at the frame.
				WriteIdent(want, NewIdentManager(), []byte("a"))
				WriteStop(want)
				if !bytes.Equal(got, want.Bytes()) {
					return fmt.Errorf("expected: %v, got: %v", byteutil.ToUpperHex(want.Bytes()), byteutil.ToUpperHex(got))
				}
				return nil
			},
		},
		{
			"callable_scope_with_bool",
			`ident a = { true; };`,
			//nolint:errcheck
			func(got []byte) error {
				want := bytes.NewBuffer(make([]byte, 0))
				WritePush(want, byteutil.TrueTape(byteutil.DefaultTapeSize), byteutil.DefaultTapeSize)
				WriteAnswer(want)
				// The scope reads no position, so its names begin at the frame.
				WriteIdent(want, NewIdentManager(), []byte("a"))
				WriteStop(want)
				if !bytes.Equal(got, want.Bytes()) {
					return fmt.Errorf("expected: %v, got: %v", byteutil.ToUpperHex(want.Bytes()), byteutil.ToUpperHex(got))
				}
				return nil
			},
		},
		{
			"callable_scope_with_feed",
			`ident a = { feed(0) - feed(1); };`,
			//nolint:errcheck
			func(got []byte) error {
				want := bytes.NewBuffer(make([]byte, 0))
				WriteGetArg(want, byteutil.FromUint64(1), byteutil.DefaultTapeSize)
				WriteGetArg(want, byteutil.FromUint64(0), byteutil.DefaultTapeSize)
				WriteSubtract(want)
				WriteMask(want, byteutil.DefaultTapeSize)
				WriteAnswer(want)
				// The scope reads two positions, so its names begin past them.
				WriteIdent(want, NewIdentManagerAt(2*MEMORY_SLOT_SIZE), []byte("a"))
				WriteStop(want)
				if !bytes.Equal(got, want.Bytes()) {
					return fmt.Errorf("expected: %v, got: %v", byteutil.ToUpperHex(want.Bytes()), byteutil.ToUpperHex(got))
				}
				return nil
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			bs := bytes.NewBufferString(c.SourceCode).Bytes()

			tokens, err := lexer.New().GetFilledTokens(bs)
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			tree, err := parser.New().Parse(parser.ParseInput{Tokens: tokens})
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			rc, err := NewBuilder(insts, NewBuilderOptions{}).PickRuntimeCode()
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			got := rc.Root.Bytes()
			if err := c.FnExpected(got); err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
		})
	}
}

func TestPickDeferAtCursor(t *testing.T) {
	cases := []struct {
		Name           string
		Insts          []ir.Instruction
		Cursor         int
		WantOK         bool
		WantNextCursor int
		WantSelector   string // only checked when WantOK
		WantBodyLength int    // only checked when WantOK
	}{
		{
			Name: "valid_defer",
			Insts: []ir.Instruction{
				ir.NewInstruction([]byte("0"), ir.OpDefer, ir.RefTo([]byte("ret")), ir.TargetAt(2)),
				ir.NewInstruction([]byte("1"), ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
				ir.NewInstruction([]byte("2"), ir.OpReturn, ir.RefTo(nil), ir.RefTo(nil)),
				ir.NewInstruction([]byte("3"), ir.OpIdent, ir.NameOf("f"), ir.RefTo([]byte("0"))),
			},
			Cursor:         0,
			WantOK:         true,
			WantNextCursor: 3,
			WantSelector:   "f",
			WantBodyLength: 2,
		},
		{
			Name: "not_op_defer",
			Insts: []ir.Instruction{
				ir.NewInstruction(nil, ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
			},
			Cursor:         0,
			WantOK:         false,
			WantNextCursor: 0,
		},
		{
			Name: "defer_without_op_ident_after",
			Insts: []ir.Instruction{
				ir.NewInstruction(nil, ir.OpDefer, ir.RefTo(nil), ir.TargetAt(2)),
				ir.NewInstruction(nil, ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
				ir.NewInstruction(nil, ir.OpReturn, ir.RefTo(nil), ir.RefTo(nil)),
				ir.NewInstruction(nil, ir.OpAdd, ir.RefTo(nil), ir.RefTo(nil)),
			},
			Cursor:         0,
			WantOK:         false,
			WantNextCursor: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			b := NewBuilder(c.Insts, NewBuilderOptions{})
			d, nextCursor, ok := b.PickDeferAtCursor(c.Cursor)
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
			if len(d.Body) != c.WantBodyLength {
				t.Errorf("body is %d instructions, want %d", len(d.Body), c.WantBodyLength)
			}
			// Finding a scope does not write it: where it lands is not known yet.
			if d.Code != nil {
				t.Error("a scope was written while it was being found")
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
	if n := m.GetLength(); n != 2 {
		t.Errorf("after two SetOffset, GetLength = %d, want 2", n)
	}

	t.Run("set_offset_overwrite", func(t *testing.T) {
		m.SetOffset("x", 0)
		m.SetOffset("x", 64)
		if got := m.GetOffset([]byte("x")); got != 64 {
			t.Errorf("GetOffset(x) after overwrite = %d, want 64", got)
		}
	})
}

// A scope written inside another is taken out of the body before it is written, at every
// depth, and what stays in its place is the neutral value the binding is worth on a chain.
func TestWithoutNestedScopes(t *testing.T) {
	// outer { inner { deep { } } ; 2 }, as the emitter lays it out: a scope carries how long
	// its body is, and a scope inside it lives inside those instructions.
	insts := []ir.Instruction{
		ir.NewInstruction([]byte("0"), ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
		ir.NewInstruction([]byte("1"), ir.OpDefer, ir.RefTo([]byte("2")), ir.TargetAt(5)),
		ir.NewInstruction([]byte("2"), ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
		ir.NewInstruction([]byte("3"), ir.OpDefer, ir.RefTo([]byte("4")), ir.TargetAt(2)),
		ir.NewInstruction([]byte("4"), ir.OpBeginScope, ir.Nothing(), ir.Nothing()),
		ir.NewInstruction([]byte("5"), ir.OpReturn, ir.RefTo([]byte("4")), ir.RefTo([]byte("4"))),
		ir.NewInstruction([]byte("6"), ir.OpReturn, ir.RefTo([]byte("2")), ir.RefTo([]byte("2"))),
		ir.NewInstruction([]byte("7"), ir.OpIdent, ir.NameOf("inner"), ir.RefTo([]byte("1"))),
		ir.NewInstruction([]byte("8"), ir.OpSave, ir.Imm(2, 8), ir.Nothing()),
		ir.NewInstruction([]byte("9"), ir.OpReturn, ir.RefTo([]byte("0")), ir.RefTo([]byte("8"))),
	}

	kept := withoutNestedScopes(insts, 8)

	wanted := []byte{ir.OpBeginScope, ir.OpSave, ir.OpIdent, ir.OpSave, ir.OpReturn}
	if len(kept) != len(wanted) {
		t.Fatalf("kept %d instructions, want %d: %v", len(kept), len(wanted), kept)
	}
	for at, want := range wanted {
		if got := kept[at].GetOpCode(); got != want {
			t.Errorf("instruction %d is %s, want %s", at, ir.ResolveOpCode(got), ir.ResolveOpCode(want))
		}
	}

	// The binding still finds what it was bound to, so nothing that read the scope's label is
	// left pointing at an instruction that is no longer there.
	if got := string(kept[1].GetLabel()); got != "1" {
		t.Errorf("what replaced the scope answers to %q, want the label the binding reads", got)
	}
}
