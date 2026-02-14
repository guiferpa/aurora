package evaluator

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/environ"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

func TestEvaluateAdd(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(1))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(2))
	if err := ev.EvaluateAdd([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating add: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.FromUint64(3)
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateSubtract(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateSubtract([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating subtract: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.FromUint64(1)
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateMultiply(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateMultiply([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating multiply: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.FromUint64(2)
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateDivide(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateDivide([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating divide: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.FromUint64(2)
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateExponential(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(3))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(3))
	if err := ev.EvaluateExponential([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating exponential: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.FromUint64(27)
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateDiff(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateDiff([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating diff: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.True
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateEquals(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateEquals([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating equals: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.False
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateBigger(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateBigger([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating bigger: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.True
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateSmaller(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(2))
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(1))
	if err := ev.EvaluateSmaller([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating smaller: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.False
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateAnd(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.True)
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.True)
	if err := ev.EvaluateAnd([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating and: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.True
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateAnd_False(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.True)
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.False)
	if err := ev.EvaluateAnd([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating and: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.False
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateOr(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.False)
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.True)
	if err := ev.EvaluateOr([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating or: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.True
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateOr_False(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.False)
	ev.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.False)
	if err := ev.EvaluateOr([]byte("02"), []byte("00"), []byte("01")); err != nil {
		t.Errorf("Error evaluating or: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	expected := byteutil.False
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateSave(t *testing.T) {
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
	})
	val := byteutil.FromUint64(42)
	if err := ev.EvaluateSave([]byte("00"), val, nil); err != nil {
		t.Errorf("Error evaluating save: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("00")))
	if !bytes.Equal(got, val) {
		t.Errorf("got: %v, expected: %v", got, val)
	}
}

func TestEvaluateLoad(t *testing.T) {
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
	})
	expected := byteutil.FromUint64(13)
	ev.environ.SetIdent(byteutil.ToHex([]byte("00")), expected)
	if err := ev.EvaluateLoad([]byte("01"), []byte("00"), nil); err != nil {
		t.Errorf("Error evaluating load: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("01")))
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateReturn(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	ev.environ = ev.environ.Ahead(next)

	expected := byteutil.FromUint64(100)
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), expected)
	if err := ev.EvaluateReturn([]byte("01"), []byte("02"), []byte("00")); err != nil {
		t.Errorf("Error evaluating return: %v", err)
		return
	}
	// After EvaluateReturn, evaluator did Back(): ev.environ is now the caller (previous) scope, and the value was stored there at label "02".
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("02")))
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluateGetArg(t *testing.T) {
	// Args are 32 bytes each; one slot = 32 bytes
	args := make([]byte, 32)
	binary.BigEndian.PutUint64(args[24:], 77) // last 8 bytes = 77
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
		Args:          args,
	})
	if err := ev.EvaluateGetArg([]byte("00"), byteutil.FromUint64(0), nil); err != nil {
		t.Errorf("Error evaluating get arg: %v", err)
		return
	}
	got := ev.environ.GetTemp(byteutil.ToHex([]byte("00")))
	expected := args
	if !bytes.Equal(got, expected) {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestEvaluatePushArg(t *testing.T) {
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
		Args:          make([]byte, 0),
	})
	label := []byte("00")
	val := byteutil.FromUint64(99)
	ev.environ.SetTemp(byteutil.ToHex(label), val)
	if err := ev.EvaluatePushArg([]byte("01"), byteutil.FromUint64(0), label); err != nil {
		t.Errorf("Error evaluating push arg: %v", err)
		return
	}
	v := ev.environ.GetArgument(0)
	if !bytes.Equal(v, val) {
		t.Errorf("got: %v, expected: %v", v, val)
	}
}

func TestEvaluateIf(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		ev := New(NewEvaluatorOptions{EnableLogging: false})
		ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.True)
		ev.cursor = 0
		if err := ev.EvaluateIf([]byte("01"), []byte("00"), byteutil.FromUint64(10)); err != nil {
			t.Errorf("Error evaluating if: %v", err)
			return
		}
		if ev.cursor != 1 {
			t.Errorf("when condition is true cursor should be 1, got: %d", ev.cursor)
		}
	})

	t.Run("False", func(t *testing.T) {
		ev := New(NewEvaluatorOptions{EnableLogging: false})
		ev.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.False)
		ev.cursor = 0
		if err := ev.EvaluateIf([]byte("01"), []byte("00"), byteutil.FromUint64(3)); err != nil {
			t.Errorf("Error evaluating if: %v", err)
			return
		}
		if ev.cursor != 4 {
			t.Errorf("when condition is false cursor should be 0+3+1=4, got: %d", ev.cursor)
		}
	})
}

func TestEvaluateJump(t *testing.T) {
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
	})
	ev.cursor = 0
	if err := ev.EvaluateJump([]byte("00"), byteutil.FromUint64(2), nil); err != nil {
		t.Errorf("Error evaluating jump: %v", err)
		return
	}
	if ev.cursor != 3 {
		t.Errorf("after jump(2) cursor should be 0+2+1=3, got: %d", ev.cursor)
	}
}

func TestEvaluatePrint(t *testing.T) {
	var buf bytes.Buffer
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
		PrintWriter:   &buf,
	})
	val := byteutil.FromUint64(7)
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), val)
	if err := ev.EvaluatePrint([]byte("01"), []byte("00")); err != nil {
		t.Errorf("Error evaluating print: %v", err)
		return
	}
	if buf.Len() == 0 {
		t.Errorf("expected something written to print writer")
	}
}

func TestEvaluateEcho(t *testing.T) {
	var buf bytes.Buffer
	ev := New(NewEvaluatorOptions{
		EnableLogging: false,
		EchoWriter:    &buf,
	})
	val := []byte("hello")
	ev.environ.SetTemp(byteutil.ToHex([]byte("00")), val)
	if err := ev.EvaluateEcho([]byte("01"), []byte("00")); err != nil {
		t.Errorf("Error evaluating echo: %v", err)
		return
	}
	if buf.Len() == 0 {
		t.Errorf("expected something written to echo writer")
	}
}

func TestCanReadInstructions(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	ev.SetInstructions([]emitter.Instruction{
		emitter.NewInstruction([]byte("00"), emitter.OpSave, nil, nil),
		emitter.NewInstruction([]byte("01"), emitter.OpAdd, nil, nil),
	})
	ev.SetInstructionsOffset(0, 2)

	if !ev.CanReadInstructions() {
		t.Error("expected CanReadInstructions true when cursor 0 < end 2")
	}
	ev.IncrementCursor()
	if !ev.CanReadInstructions() {
		t.Error("expected CanReadInstructions true when cursor 1 < end 2")
	}
	ev.IncrementCursor()
	if ev.CanReadInstructions() {
		t.Error("expected CanReadInstructions false when cursor 2 >= end 2")
	}
}

func TestGetInstruction(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	inst0 := emitter.NewInstruction([]byte("00"), emitter.OpSave, []byte{1}, nil)
	inst1 := emitter.NewInstruction([]byte("01"), emitter.OpAdd, []byte("00"), []byte("01"))
	ev.SetInstructions([]emitter.Instruction{inst0, inst1})
	ev.SetInstructionsOffset(0, 2)

	got := ev.GetInstruction()
	if got.GetOpCode() != emitter.OpSave || !bytes.Equal(got.GetLabel(), []byte("00")) {
		t.Errorf("expected first instruction (OpSave 00), got op=%d label=%s", got.GetOpCode(), got.GetLabel())
	}
	ev.IncrementCursor()
	got = ev.GetInstruction()
	if got.GetOpCode() != emitter.OpAdd || !bytes.Equal(got.GetLabel(), []byte("01")) {
		t.Errorf("expected second instruction (OpAdd 01), got op=%d label=%s", got.GetOpCode(), got.GetLabel())
	}
}

func TestSetInstructions(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	insts := []emitter.Instruction{
		emitter.NewInstruction([]byte("00"), emitter.OpSave, nil, nil),
	}
	ev.SetInstructions(insts)
	ev.SetInstructionsOffset(0, uint64(len(insts)))

	got := ev.GetInstruction()
	if got.GetOpCode() != emitter.OpSave {
		t.Errorf("expected OpSave after SetInstructions, got op=%d", got.GetOpCode())
	}
}

func TestSetInstructionsOffset(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	ev.SetInstructions([]emitter.Instruction{
		emitter.NewInstruction([]byte("00"), emitter.OpSave, nil, nil),
		emitter.NewInstruction([]byte("01"), emitter.OpSave, nil, nil),
		emitter.NewInstruction([]byte("02"), emitter.OpAdd, nil, nil),
	})
	ev.SetInstructionsOffset(1, 3)

	cursor, end := ev.GetInstructionsOffset()
	if cursor != 1 || end != 3 {
		t.Errorf("expected cursor=1 end=3, got cursor=%d end=%d", cursor, end)
	}
	got := ev.GetInstruction()
	if !bytes.Equal(got.GetLabel(), []byte("01")) {
		t.Errorf("expected instruction at offset 1 (label 01), got label=%s", got.GetLabel())
	}
}

func TestGetInstructionsOffset(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	ev.SetInstructionsOffset(2, 5)
	cursor, end := ev.GetInstructionsOffset()
	if cursor != 2 || end != 5 {
		t.Errorf("expected cursor=2 end=5, got cursor=%d end=%d", cursor, end)
	}
}

func TestIncrementCursor(t *testing.T) {
	ev := New(NewEvaluatorOptions{EnableLogging: false})
	ev.SetInstructionsOffset(0, 3)

	ev.IncrementCursor()
	cursor, _ := ev.GetInstructionsOffset()
	if cursor != 1 {
		t.Errorf("expected cursor 1 after IncrementCursor, got %d", cursor)
	}
	ev.IncrementCursor()
	cursor, _ = ev.GetInstructionsOffset()
	if cursor != 2 {
		t.Errorf("expected cursor 2 after second IncrementCursor, got %d", cursor)
	}
}

func TestAddCursor(t *testing.T) {
	ev := New(NewEvaluatorOptions{})
	ev.SetInstructionsOffset(0, 10)

	ev.AddCursor(3)
	cursor, _ := ev.GetInstructionsOffset()
	if cursor != 3 {
		t.Errorf("expected cursor 3 after AddCursor(3), got %d", cursor)
	}
	ev.AddCursor(4)
	cursor, _ = ev.GetInstructionsOffset()
	if cursor != 7 {
		t.Errorf("expected cursor 7 after AddCursor(4), got %d", cursor)
	}
}

type EvaluateCase struct {
	Name       string
	SourceCode string
	TestFn     func(t *testing.T, returns ReturnsPerLabel, err error)
}

type RunEvaluateCaseOptions struct {
	Filename      string
	EnableLogging bool
}

func runEvaluateCase(t *testing.T, cases []EvaluateCase, options RunEvaluateCaseOptions) {
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			bs := bytes.NewBufferString(c.SourceCode).Bytes()

			tokens, err := lexer.New(lexer.NewLexerOptions{
				EnableLogging: false,
			}).GetFilledTokens(bs)
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}

			ast, err := parser.New(tokens, parser.NewParserOptions{
				Filename:      options.Filename,
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

			ev := New(NewEvaluatorOptions{
				EnableLogging: options.EnableLogging,
			})
			m, err := ev.Evaluate(insts)

			if c.TestFn != nil {
				c.TestFn(t, m, err)
			} else if err != nil {
				t.Errorf("%v: %v", c.Name, err)
			}
		})
	}
}

func TestRelative(t *testing.T) {
	cases := []EvaluateCase{
		{
			"relative_1",
			`1 different 2;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 different 2" the emitter generates Save(00), Save(01), Different(02). The Different stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_2",
			`1 equals 2;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 equals 2" the emitter generates Save(00), Save(01), Equals(02). The Equals stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_3",
			`1 smaller 2;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 smaller 2" the emitter generates Save(00), Save(01), Smaller(02). The Smaller stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_4",
			`1 bigger 2;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 bigger 2" the emitter generates Save(00), Save(01), Bigger(02). The Bigger stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_5",
			`1 equals 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 equals 1" the emitter generates Save(00), Save(01), Equals(02). The Equals stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_6",
			`1 different 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 different 1" the emitter generates Save(00), Save(01), Different(02). The Different stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_7",
			`1 smaller 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 smaller 1" the emitter generates Save(00), Save(01), Smaller(02). The Smaller stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
				expected = []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"relative_8",
			`1 bigger 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 bigger 1" the emitter generates Save(00), Save(01), Bigger(02). The Bigger stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}
	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestArithmetic(t *testing.T) {
	cases := []EvaluateCase{
		{
			"arithmetic_1",
			`1 + 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 + 1" the emitter generates Save(00), Save(01), Add(02). The Add stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 2}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"arithmetic_2",
			`1 - 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 - 1" the emitter generates Save(00), Save(01), Sub(02). The Sub stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"arithmetic_3",
			`1 * 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 * 1" the emitter generates Save(00), Save(01), Mul(02). The Mul stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"arithmetic_4",
			`1 / 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 / 1" the emitter generates Save(00), Save(01), Div(02). The Div stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"arithmetic_5",
			`1 ^ 1;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "1 ^ 1" the emitter generates Save(00), Save(01), Exp(02). The Exp stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestBoolean(t *testing.T) {
	cases := []EvaluateCase{
		{
			"boolean_1",
			`true or false;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "true or false" the emitter generates Save(00), Save(01), Or(02). The Or stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"boolean_2",
			`false or false;
      true and true;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 02: for "false or false" the emitter generates Save(00), Save(01), Or(02). The Or stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label := byteutil.ToHex([]byte("02"))
				got := returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected := []byte{0}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}

				// Label 05: for "true and true" the emitter generates Save(03), Save(04), And(05). The And stores its result in its own label (no OpResult), so the result 2 lives at temp "02".
				label = byteutil.ToHex([]byte("05"))
				got = returns[label]
				if len(got) != 1 {
					t.Errorf("Boolean value should be 1 byte, got: %v", got)
				}
				expected = []byte{1}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestIfAndElse(t *testing.T) {
	cases := []EvaluateCase{
		{
			"if_1",
			`if 10 bigger 9 { 10; };
if 11 bigger 10 { 20; };`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				// Label 01: for "if 10 bigger 9 { 10; };" the emitter generates Save(00), Save(01), If(02). The If stores its result in its own label (no OpResult), so the result 1 lives at temp "01".
				label := byteutil.ToHex([]byte("04"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 10}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}

				// Second if result: the emitter uses a single global label counter; the first if consumes labels 00–07 (or so), so the second if's result lands at label 10, formatted as "010" by GenerateLabel ("0%d").
				label = byteutil.ToHex([]byte("012"))
				got = returns[label]
				expected = []byte{0, 0, 0, 0, 0, 0, 0, 20}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"if_with_else_1",
			`if 10 bigger 9 { 10; } else { 20; };`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("05"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 10}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"if_with_else_2",
			`if 10 bigger 11 { 10; } else { 20; };`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("05"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 20}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}
	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestBranch(t *testing.T) {
	cases := []EvaluateCase{
		// Parse: branch is reduced to nested if/else. getBranchItem() parses "cond : value , next"
		// as IfExpressionNode{ Test: cond, Body: [value], Else: &ElseExpressionNode{ Body: [next] } },
		// and "expr;" as the default (no condition), returning just expr. So this branch becomes:
		//   if (op equals 1) { 32 } else { if (op equals 2) { 64 } else { 128 } }
		{
			"branch_1",
			`ident op = 2;
branch {
  op equals 1: 32, 
  op equals 2: 64, 
  128;
};`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("015"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 64}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"branch_2",
			`ident op = 2;
ident r = branch {
  op equals 1: 32, 
  op equals 2: 64, 
  128;
};
r;`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("020"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 64}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestCallableScope(t *testing.T) {
	cases := []EvaluateCase{
		{
			"callable_scope_1",
			`{ 1 + 2; };`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("03"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 3}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"callable_scope_2",
			`{ 1 + { 2 + 3; }; };`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("07"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 6}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"callable_scope_3",
			`{ 
  ident a = 1;
  {
    ident a = 2;
    a * 2;
  };
};`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("09"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 4}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"callable_scope_4",
			`{ 
  ident a = 1;
  {
    a * 2;
  };
};`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("07"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 2}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestDefer(t *testing.T) {
	cases := []EvaluateCase{
		{
			"defer_1",
			`ident r = defer {
  arguments 0 + arguments 1;
};

r(1, 2);
r(3, 4);`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("011"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 3}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}

				label = byteutil.ToHex([]byte("016"))
				got = returns[label]
				expected = []byte{0, 0, 0, 0, 0, 0, 0, 7}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

func TestDeferRecursivity(t *testing.T) {
	cases := []EvaluateCase{
		{
			"fibonacci_1",
			`ident fib = defer {
  ident n = arguments 0;
  if n smaller 1 or n equals 1 { n; } else { fib(n - 1) + fib(n - 2); };
};

fib(11);`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("031"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 89}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"fibonacci_2",
			`ident fib = defer {
  ident n = arguments 0;
  branch {
    n smaller 1 or n equals 1: n,
    fib(n - 1) + fib(n - 2);
  };
};

fib(11);`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("031"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 89}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
		{
			"factorial_1",
			`ident factorial = defer {
  ident n = arguments 0;
  if n smaller 1 or n equals 1 { 1; } else { n * factorial(n - 1); };
};

factorial(4);`,
			func(t *testing.T, returns ReturnsPerLabel, err error) {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				label := byteutil.ToHex([]byte("027"))
				got := returns[label]
				expected := []byte{0, 0, 0, 0, 0, 0, 0, 24}
				if !bytes.Equal(got, expected) {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	runEvaluateCase(t, cases, RunEvaluateCaseOptions{})
}

type AssertCase struct {
	Name       string
	SourceCode string
	TestFn     func(t *testing.T, errors []error)
}

func runAssertCase(t *testing.T, cases []AssertCase) {
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			bs := bytes.NewBufferString(c.SourceCode).Bytes()
			tokens, err := lexer.New(lexer.NewLexerOptions{
				EnableLogging: false,
			}).GetFilledTokens(bs)
			if err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			ast, err := parser.New(tokens, parser.NewParserOptions{
				Filename:      ".test.ar",
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
			ev := New(NewEvaluatorOptions{
				EnableLogging: false,
			})
			if _, err := ev.Evaluate(insts); err != nil {
				t.Errorf("%v: %v", c.Name, err)
				return
			}
			if c.TestFn != nil {
				c.TestFn(t, ev.GetAssertErrors())
			}
		})
	}
}

func TestAssert(t *testing.T) {
	cases := []AssertCase{
		{
			"assert_equals_pass",
			`assert(2 equals 2, "expected 2 to equal 2");`,
			func(t *testing.T, errors []error) {
				if len(errors) != 0 {
					t.Errorf("expected no errors, got: %v", errors)
				}
			},
		},
		{
			"assert_equals_fail",
			`assert(1 equals 2, "expected 1 to equal 2");`,
			func(t *testing.T, errors []error) {
				if len(errors) != 1 {
					t.Errorf("expected 1 error, got: %v", errors)
				}
				err := errors[0]
				if err.Error() != "assertion failed: expected 1 to equal 2" {
					t.Errorf("expected error message to contain custom message, got: %v", errors[0])
				}
			},
		},
		{
			"assert_with_variable",
			`ident a = 10;
assert(a equals 11, "a should be 10");`,
			func(t *testing.T, errors []error) {
				if len(errors) != 1 {
					t.Errorf("expected 1 error, got: %v", errors)
				}
				err := errors[0]
				if err.Error() != "assertion failed: a should be 10" {
					t.Errorf("expected error message to contain custom message, got: %v", errors[0])
				}
			},
		},
		{
			"assert_with_expression",
			`assert(2 + 4 equals 4, "2 + 4 = 6");`,
			func(t *testing.T, errors []error) {
				if len(errors) != 1 {
					t.Errorf("expected 1 error, got: %v", errors)
				}
				err := errors[0]
				if err.Error() != "assertion failed: 2 + 4 = 6" {
					t.Errorf("expected error message to contain custom message, got: %v", errors[0])
				}
			},
		},
		{
			"assert_with_function_call",
			`ident sum = defer {
  ident x = arguments 0;
  ident y = arguments 1;
  x + y;
};

assert(sum(2, 3) equals 5, "sum(2, 3) should be 5");`,
			func(t *testing.T, errors []error) {
				if len(errors) != 0 {
					t.Errorf("expected no errors, got: %v", errors)
				}
			},
		},
	}

	runAssertCase(t, cases)
}
