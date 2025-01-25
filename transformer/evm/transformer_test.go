package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

type mockWriter struct {
	Buffer []byte
}

func (mw *mockWriter) Write(bs []byte) (int, error) {
	mw.Buffer = append(mw.Buffer, bs...)
	return 0, nil
}

func TestTransform(t *testing.T) {
	cases := []struct {
		Name       string
		SourceCode string
		Fn         func(string, *mockWriter) func(t *testing.T)
	}{
		{
			"math_sum_1",
			`10 + 20_001;`,
			func(name string, mw *mockWriter) func(t *testing.T) {
				return func(t *testing.T) {
					expected := []byte{}
					if got, expected := mw.Buffer, expected; !bytes.Equal(got, expected) {
						t.Errorf("EVM transformer: name: %v, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
	}

	transformer := &Transformer{}

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
		mw := &mockWriter{Buffer: make([]byte, 0)}
		if err := transformer.Transform(mw, insts); err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		t.Run(c.Name, c.Fn(c.Name, mw))
	}
}
