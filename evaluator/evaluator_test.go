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
					if got, expected := r[1], []byte{0, 0, 0, 0, 0, 0, 0, 2}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[2], []byte{0, 0, 0, 0, 0, 0, 0, 40}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 1, 144}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_1",
			"tape 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_append_1",
			`ident target = tape 3;
      ident t1 = append target 1;
      ident t2 = append t1 2;
      append t2 3;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_head_1",
			`ident target = tape 3;
      ident t1 = append target 1;
      ident t2 = append t1 2;
      ident t3 = append t2 3;
      head t3 2;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_tail_1",
			`ident target = tape 3;
      ident t1 = append target 1;
      ident t2 = append t1 2;
      ident t3 = append t2 3;
      tail t3 2;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_push_1",
			`ident target = tape 3;
      ident t1 = append target 1;
      ident t2 = append t1 2;
      push t2 3;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}; !bytes.Equal(got, expected) {
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
		{
			"tape_bracket_2",
			"[];",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					expected := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
					if got, expected := r[0], expected; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_append_1",
			"append [1, 2] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_head_1",
			"head [1, 2, 3] 2;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_tail_1",
			`tail [1, 2, 3] 2;`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_push_1",
			"push [1, 2] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 1}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_push_2",
			"push [1, 2, 3] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2}; !bytes.Equal(got, expected) {
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
		m, err := New(false).Evaluate(insts)
		if err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		r := make([][]byte, 0)
		for _, k := range slices.Sorted(maps.Keys(m)) {
			r = append(r, m[k])
		}
		t.Run(c.Name, c.Fn(c.Name, r))
	}
}
