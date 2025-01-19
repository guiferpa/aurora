package evaluator

import (
	"bytes"
	"maps"
	"slices"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func TestIsLabel(t *testing.T) {
	values := [][]byte{
		[]byte("0t"),
		[]byte("0-1t"),
	}
	for _, v := range values {
		if !isTemp(v) {
			t.Errorf("unrecognized as label pattern, got: %s", v)
			return
		}
	}
}

func TestEvaluate(t *testing.T) {
	cases := []struct {
		Name       string
		SourceCode string
		Fn         func(string, [][]byte) func(t *testing.T)
	}{
		{
			"boolean_1",
			`true or false;
      false and true;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{1}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[1], []byte{0}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"boolean_2",
			`false or false;
      true and true;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[1], []byte{1}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"math_1",
			`1 + 1;
      20 + 20;
      200 + 2_00;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 2}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[1], []byte{0, 0, 0, 0, 0, 0, 0, 40}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[2], []byte{0, 0, 0, 0, 0, 0, 1, 144}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_1",
			"[1, 20, 300];",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					expected := []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 20, 0, 0, 0, 0, 0, 0, 1, 44}
					if got, expected := r[0], expected; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
	}
	for _, c := range cases {
		bs := bytes.NewBufferString(c.SourceCode).Bytes()
		tokens, err := lexer.GetFilledTokens(bs)
		if err != nil {
			t.Error(err)
		}
		ast, err := parser.New(tokens).Parse()
		if err != nil {
			t.Error(err)
		}
		insts, err := emitter.New().Emit(ast)
		if err != nil {
			t.Error(err)
		}
		m, err := New(false).Evaluate(insts)
		if err != nil {
			t.Error(err)
		}
		t.Run(c.Name, c.Fn(c.Name, slices.Collect[[]byte](maps.Values(m))))
	}
}
