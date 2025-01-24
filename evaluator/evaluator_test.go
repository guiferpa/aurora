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
			"relative_1",
			`1 different 2;
      2 different 2;`,
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
			"relative_1",
			`1 equals 2;
      2 equals 2;`,
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
			"if_1",
			`if 10 bigger 9 {
        10;
      };

      if 11 bigger 10 {
        20;
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 20}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
					if got, expected := r[1], []byte{0, 0, 0, 0, 0, 0, 0, 10}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"if_with_else_1",
			"if 9 bigger 9 { 90; } else { 100; };",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 100}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"if_with_else_2",
			"if 10 bigger 9 { 90; } else { 100; };",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 90}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"if_with_else_3",
			`ident op = 2;
      if op equals 1 { 1 + 1; } else { 
        if op equals 2 { 1 - 1; } else { 10; };
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 0}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"if_with_else_4",
			`ident op = 3;
      if op equals 1 { 1 + 1; } else { 
        if op equals 2 { 1 - 1; } else { 10; };
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 10}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"callable_scope_1",
			`ident fn = {
        ident r = 1 + 2;
        r; 
      };

      fn();`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"callable_scope_with_arguments_1",
			`ident fn = {
        ident x = arguments 0;
        ident y = arguments 1;
        x + y;
      };

      fn(10, 50);`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 60}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"branch_1",
			`ident sum = {
        ident a = arguments 0;
        ident b = arguments 1;
        a + b;
      };

      ident sub = {
        ident a = arguments 0;
        ident b = arguments 1;
        a - b;
      };

      ident op = 2;

      branch {
        op equals 1: sum(10, 1), 
        op equals 2: sub(10, 1), 
        10;
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 9}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"branch_2",
			`ident sum = {
        ident a = arguments 0;
        ident b = arguments 1;
        a + b;
      };

      ident sub = {
        ident a = arguments 0;
        ident b = arguments 1;
        a - b;
      };

      ident op = 1;

      branch {
        op equals 1: sum(10, 1), 
        op equals 2: sub(10, 1), 
        10;
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 11}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"branch_3",
			`ident sum = {
        ident a = arguments 0;
        ident b = arguments 1;
        a + b;
      };

      ident sub = {
        ident a = arguments 0;
        ident b = arguments 1;
        a - b;
      };

      ident op = 3;

      branch {
        op equals 1: sum(10, 1), 
        op equals 2: sub(10, 1), 
        10;
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 10}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"branch_4",
			`ident sum = {
        ident a = arguments 0;
        ident b = arguments 1;
        a + b;
      };

      ident sub = {
        ident a = arguments 0;
        ident b = arguments 1;
        a - b;
      };

      ident op = 3;

      ident another_op = false;

      branch {
        op equals 1: sum(1, 1), 
        op equals 2: sub(1, 1),
        branch {
          another_op: 10,
          12;
        };
      };`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 12}; !bytes.Equal(got, expected) {
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
			"tape_pull_1",
			`ident target = tape 3;
      ident t1 = pull target 1;
      ident t2 = pull t1 2;
      pull t2 3;`,
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
      ident t1 = pull target 1;
      ident t2 = pull t1 2;
      ident t3 = pull t2 3;
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
      ident t1 = pull target 1;
      ident t2 = pull t1 2;
      ident t3 = pull t2 3;
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
      ident t1 = pull target 1;
      ident t2 = pull t1 2;
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
			"tape_bracket_pull_1",
			"pull [1, 2] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"tape_bracket_pull_2",
			"pull [] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}; !bytes.Equal(got, expected) {
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
		{
			"tape_bracket_push_3",
			"push [] 3;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"fibonacci_1",
			`ident fib = {
        ident n = arguments 0;
        if n smaller 1 or n equals 1 { n; } else { fib(n - 1) + fib(n - 2); };
      };

      fib(11);`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 89}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"fibonacci_2",
			`ident fib = {
        ident n = arguments 0;
        branch {
          n smaller 1 or n equals 1: n,
          fib(n - 1) + fib(n - 2);
        };
      };

      fib(11);`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 89}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"factorial_1",
			`ident factorial = {
        ident n = arguments 0;
        if n smaller 1 or n equals 1 { 1; } else { n * factorial(n - 1); };
      };

      factorial(4);`,
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 24}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"glue_1",
			"glue [1, 2] [2, 1];",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1}; !bytes.Equal(got, expected) {
						t.Errorf("%s, got: %v, expected: %v", name, got, expected)
					}
				}
			},
		},
		{
			"glue_2",
			"glue 13 12;",
			func(name string, r [][]byte) func(t *testing.T) {
				return func(t *testing.T) {
					if got, expected := r[0], []byte{0, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 0, 0, 0, 0, 12}; !bytes.Equal(got, expected) {
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
