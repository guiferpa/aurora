package evm

import (
	"bytes"
	"fmt"
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
				// Three blocks: the run up to the block written inside, the block itself,
				// and where the run carries on with its value in hand.
				want.Write([]byte{OpJumpDestiny, OpJumpDestiny})
				WritePush(want, byteutil.FromUint64(4294967295), byteutil.DefaultTapeSize)
				WritePush(want, byteutil.FromUint64(4294967295), byteutil.DefaultTapeSize)
				WriteAdd(want)
				WriteMask(want, byteutil.DefaultTapeSize)
				want.Write([]byte{OpJumpDestiny})
				WriteIdent(want, map[string]int{"a": 0}, []byte("a"))
				WritePush(want, nil, byteutil.DefaultTapeSize)
				WriteReturnToChain(want)
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
				// Three blocks: the run up to the block written inside, the block itself,
				// and where the run carries on with its value in hand.
				want.Write([]byte{OpJumpDestiny, OpJumpDestiny})
				WritePush(want, byteutil.TrueTape(byteutil.DefaultTapeSize), byteutil.DefaultTapeSize)
				want.Write([]byte{OpJumpDestiny})
				WriteIdent(want, map[string]int{"a": 0}, []byte("a"))
				WritePush(want, nil, byteutil.DefaultTapeSize)
				WriteReturnToChain(want)
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
				// Three blocks: the run up to the block written inside, the block itself,
				// and where the run carries on with its value in hand.
				want.Write([]byte{OpJumpDestiny, OpJumpDestiny})
				WriteGetArg(want, byteutil.FromUint64(1), byteutil.DefaultTapeSize)
				WriteGetArg(want, byteutil.FromUint64(0), byteutil.DefaultTapeSize)
				WriteSubtract(want)
				WriteMask(want, byteutil.DefaultTapeSize)
				want.Write([]byte{OpJumpDestiny})
				WriteIdent(want, map[string]int{"a": 2 * MEMORY_SLOT_SIZE}, []byte("a"))
				WritePush(want, nil, byteutil.DefaultTapeSize)
				WriteReturnToChain(want)
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
