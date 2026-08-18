package evaluator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/evaluator/environ"
	"github.com/guiferpa/aurora/wire/ir"
)

// ExecuteInstruction is the only place that says which opcode runs which operation, and
// nothing here exercised it: every other test in this package calls EvaluateAdd and its
// neighbours by hand. An opcode pointing at the wrong operation — bigger at smaller, head at
// tail — went through the whole suite in silence.
//
// So every case below goes in through the opcode, and the operands are picked so that no two
// operations agree: six and three give five different answers to the five arithmetic
// opcodes, and each comparison is asked twice, because one question alone cannot tell "is
// different" from "is bigger".

const tapeSize = 8

type dispatchCase struct {
	name   string
	opcode byte
	setup  func(e *Evaluator)
	left   []byte
	right  []byte
	want   []byte // what label "02" holds afterwards; nil when check does the asking
	cursor uint64 // where the cursor lands, counted from zero
	check  func(t *testing.T, e *Evaluator)
}

// operands puts the two values a binary operation reads under the labels it reads them from.
func operands(x, y uint64) func(e *Evaluator) {
	return func(e *Evaluator) {
		e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(x))
		e.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(y))
	}
}

var dispatchCases = []dispatchCase{
	{
		name: "add", opcode: ir.OpAdd, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FromUint64(9), cursor: 1,
	},
	{
		name: "subtract", opcode: ir.OpSubtract, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FromUint64(3), cursor: 1,
	},
	{
		name: "multiply", opcode: ir.OpMultiply, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FromUint64(18), cursor: 1,
	},
	{
		name: "divide", opcode: ir.OpDivide, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FromUint64(2), cursor: 1,
	},
	{
		name: "exponential", opcode: ir.OpExponential, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FromUint64(216), cursor: 1,
	},

	// Asked of (6, 3) and then of (3, 6), the four comparisons answer (true, true),
	// (false, false), (true, false) and (false, true) — four different pairs, so a swap
	// between any two of them shows.
	{
		name: "diff, the bigger first", opcode: ir.OpDiff, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.TrueTape(tapeSize), cursor: 1,
	},
	{
		name: "diff, the smaller first", opcode: ir.OpDiff, setup: operands(3, 6),
		left: []byte("00"), right: []byte("01"), want: byteutil.TrueTape(tapeSize), cursor: 1,
	},
	{
		name: "equals, the bigger first", opcode: ir.OpEquals, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FalseTape(tapeSize), cursor: 1,
	},
	{
		name: "equals, the smaller first", opcode: ir.OpEquals, setup: operands(3, 6),
		left: []byte("00"), right: []byte("01"), want: byteutil.FalseTape(tapeSize), cursor: 1,
	},
	{
		name: "bigger, the bigger first", opcode: ir.OpBigger, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.TrueTape(tapeSize), cursor: 1,
	},
	{
		name: "bigger, the smaller first", opcode: ir.OpBigger, setup: operands(3, 6),
		left: []byte("00"), right: []byte("01"), want: byteutil.FalseTape(tapeSize), cursor: 1,
	},
	{
		name: "smaller, the bigger first", opcode: ir.OpSmaller, setup: operands(6, 3),
		left: []byte("00"), right: []byte("01"), want: byteutil.FalseTape(tapeSize), cursor: 1,
	},
	{
		name: "smaller, the smaller first", opcode: ir.OpSmaller, setup: operands(3, 6),
		left: []byte("00"), right: []byte("01"), want: byteutil.TrueTape(tapeSize), cursor: 1,
	},

	// and and or only differ where the operands do.
	{
		name: "and, one true one false", opcode: ir.OpAnd, setup: operands(1, 0),
		left: []byte("00"), right: []byte("01"), want: byteutil.FalseTape(tapeSize), cursor: 1,
	},
	{
		name: "or, one true one false", opcode: ir.OpOr, setup: operands(1, 0),
		left: []byte("00"), right: []byte("01"), want: byteutil.TrueTape(tapeSize), cursor: 1,
	},

	{
		name: "save", opcode: ir.OpSave,
		left: byteutil.FromUint64(7), want: byteutil.FromUint64(7), cursor: 1,
	},
	{
		name:   "load",
		opcode: ir.OpLoad,
		setup: func(e *Evaluator) {
			e.environ.SetIdent(byteutil.ToHex([]byte("id")), byteutil.FromUint64(11))
		},
		left: []byte("id"), want: byteutil.FromUint64(11), cursor: 1,
	},
	{
		name:   "ident",
		opcode: ir.OpIdent,
		setup: func(e *Evaluator) {
			e.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(5))
		},
		left: []byte("x"), right: []byte("01"), cursor: 1,
		check: func(t *testing.T, e *Evaluator) {
			got := e.environ.GetIdent(byteutil.ToHex([]byte("x")))
			if !bytes.Equal(got, byteutil.FromUint64(5)) {
				t.Errorf("x holds %v, want 5", got)
			}
		},
	},

	// if and jump both move the cursor and write nothing, so the landing is the whole answer.
	{
		name:   "if, the test holds",
		opcode: ir.OpIf,
		setup: func(e *Evaluator) {
			e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.TrueTape(tapeSize))
		},
		left: []byte("00"), right: byteutil.FromUint64(4), cursor: 1,
	},
	{
		name:   "if, the test does not hold",
		opcode: ir.OpIf,
		setup: func(e *Evaluator) {
			e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FalseTape(tapeSize))
		},
		left: []byte("00"), right: byteutil.FromUint64(4), cursor: 5,
	},
	{
		name: "jump", opcode: ir.OpJump,
		left: byteutil.FromUint64(3), cursor: 4,
	},

	{
		name:   "get feed",
		opcode: ir.OpGetFeed,
		setup: func(e *Evaluator) {
			e.environ.SetArgument(0, byteutil.FromUint64(99))
		},
		left: byteutil.FromUint64(0), want: byteutil.FromUint64(99), cursor: 1,
	},
	{
		// A defer answers with its index as a tape and steps over the body it just stored.
		name: "defer", opcode: ir.OpDefer,
		left: []byte("ret"), right: byteutil.FromUint64(2),
		want: byteutil.FalseTape(tapeSize), cursor: 3,
	},

	// The tape operations move the same two values in four different directions.
	{
		name:   "pull",
		opcode: ir.OpPull,
		setup:  operands(1, 2),
		left:   []byte("00"), right: []byte("01"),
		want: byteutil.FromUint64(0x0102), cursor: 1,
	},
	{
		name:   "push",
		opcode: ir.OpPush,
		setup:  operands(1, 2),
		left:   []byte("00"), right: []byte("01"),
		want: []byte{2, 0, 0, 0, 0, 0, 0, 0}, cursor: 1,
	},
	{
		name:   "head",
		opcode: ir.OpHead,
		setup:  operands(0x0102, 1),
		left:   []byte("00"), right: byteutil.FromUint64(1),
		want: byteutil.FromUint64(1), cursor: 1,
	},
	{
		name:   "tail",
		opcode: ir.OpTail,
		setup:  operands(0x0102, 1),
		left:   []byte("00"), right: byteutil.FromUint64(1),
		want: byteutil.FromUint64(2), cursor: 1,
	},
	{
		name:   "join",
		opcode: ir.OpJoin,
		setup:  operands(1, 2),
		left:   []byte("00"), right: []byte("01"),
		want:   append(byteutil.FromUint64(1), byteutil.FromUint64(2)...),
		cursor: 1,
	},
	{
		name:   "field",
		opcode: ir.OpField,
		setup: func(e *Evaluator) {
			run := append(byteutil.FromUint64(1), byteutil.FromUint64(2)...)
			e.environ.SetTemp(byteutil.ToHex([]byte("00")), run)
		},
		left: []byte("00"), right: byteutil.FromUint64(1),
		want: byteutil.FromUint64(2), cursor: 1,
	},

	{
		name:   "push feed",
		opcode: ir.OpPushFeed,
		setup: func(e *Evaluator) {
			e.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(42))
		},
		left: byteutil.FromUint64(0), right: []byte("01"), cursor: 1,
		check: func(t *testing.T, e *Evaluator) {
			if got := e.environ.GetArgument(0); !bytes.Equal(got, byteutil.FromUint64(42)) {
				t.Errorf("argument 0 holds %v, want 42", got)
			}
		},
	},
	{
		name: "begin scope", opcode: ir.OpBeginScope, cursor: 1,
		check: func(t *testing.T, e *Evaluator) {
			if e.environ.GetPrevious() == nil {
				t.Error("no scope was opened")
			}
		},
	},
	{
		name:   "return",
		opcode: ir.OpReturn,
		setup: func(e *Evaluator) {
			e.environ = e.environ.Ahead(environ.NewEnviron(environ.NewEnvironOptions{}))
			e.environ.SetTemp(byteutil.ToHex([]byte("01")), byteutil.FromUint64(13))
		},
		left: []byte("ret"), right: []byte("01"), cursor: 1,
		check: func(t *testing.T, e *Evaluator) {
			if got := e.environ.GetTemp(byteutil.ToHex([]byte("ret"))); !bytes.Equal(got, byteutil.FromUint64(13)) {
				t.Errorf("the caller got %v back, want 13", got)
			}
		},
	},
}

func TestExecuteInstructionDispatchesToTheRightOperation(t *testing.T) {
	for _, tc := range dispatchCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(NewEvaluatorOptions{})
			if tc.setup != nil {
				tc.setup(e)
			}

			inst := ir.NewInstruction([]byte("02"), tc.opcode, tc.left, tc.right)
			if err := e.ExecuteInstruction(inst); err != nil {
				t.Fatalf("executing %s: %v", ir.ResolveOpCode(tc.opcode), err)
			}

			if tc.want != nil {
				got := e.environ.GetTemp(byteutil.ToHex([]byte("02")))
				if !bytes.Equal(got, tc.want) {
					t.Errorf("label holds %v, want %v", got, tc.want)
				}
			}
			if cursor, _ := e.GetInstructionsOffset(); cursor != tc.cursor {
				t.Errorf("cursor landed on %d, want %d", cursor, tc.cursor)
			}
			if tc.check != nil {
				tc.check(t, e)
			}
		})
	}
}

// recorder is a printer that keeps what it was asked to print and answers with what it was
// told to answer. The evaluator no longer knows how a tape is shown — a host says that — so
// what a test here can check is which port an opcode reaches for, and what comes back.
type recorder struct {
	seen  []byte
	says  []byte
	fails error
}

func (r *recorder) Print(value []byte) ([]byte, error) {
	r.seen = value
	if r.fails != nil {
		return nil, r.fails
	}
	if r.says != nil {
		return r.says, nil
	}
	return value, nil
}

// The three prints are three readings of the same tape, so the opcode is the only thing that
// decides which one is asked.
func TestExecuteInstructionDispatchesThePrints(t *testing.T) {
	cases := []struct {
		name   string
		opcode byte
	}{
		{name: "bytes", opcode: ir.OpPrintBytes},
		{name: "chars", opcode: ir.OpPrintChars},
		{name: "decimal", opcode: ir.OpPrintDecimal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			printers := map[byte]*recorder{
				ir.OpPrintBytes:   {},
				ir.OpPrintChars:   {},
				ir.OpPrintDecimal: {},
			}
			e := New(NewEvaluatorOptions{
				PrintBytes:   printers[ir.OpPrintBytes],
				PrintChars:   printers[ir.OpPrintChars],
				PrintDecimal: printers[ir.OpPrintDecimal],
			})
			e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(65))

			inst := ir.NewInstruction([]byte("02"), tc.opcode, []byte("00"), nil)
			if err := e.ExecuteInstruction(inst); err != nil {
				t.Fatalf("executing: %v", err)
			}

			for opcode, printer := range printers {
				asked := printer.seen != nil
				if want := opcode == tc.opcode; asked != want {
					t.Errorf("%s was asked: %v, want %v", ir.ResolveOpCode(opcode), asked, want)
				}
			}
			if got := printers[tc.opcode].seen; byteutil.ToUint64(got) != 65 {
				t.Errorf("was given %v, want the value under the operand", got)
			}
		})
	}
}

// A print is an expression, so it answers with a value like everything else in Aurora — the
// one the printer gives back. Answering with nothing would make "printd x + 1" mean nothing.
func TestAPrintAnswersWithWhatThePrinterGaveBack(t *testing.T) {
	printer := &recorder{says: byteutil.FromUint64(7)}
	e := New(NewEvaluatorOptions{PrintDecimal: printer})
	e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(65))

	inst := ir.NewInstruction([]byte("02"), ir.OpPrintDecimal, []byte("00"), nil)
	if err := e.ExecuteInstruction(inst); err != nil {
		t.Fatalf("executing: %v", err)
	}

	got := e.environ.GetTemp(byteutil.ToHex([]byte("02")))
	if byteutil.ToUint64(got) != 7 {
		t.Errorf("the expression is worth %v, want what the printer answered", got)
	}
}

// A print that fails stops the program. It used to be written with "_, _ =": a program whose
// output went nowhere carried on as if it had been heard.
func TestAPrintThatFailsStopsTheProgram(t *testing.T) {
	e := New(NewEvaluatorOptions{PrintChars: &recorder{fails: errors.New("broken pipe")}})
	e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(65))

	inst := ir.NewInstruction([]byte("02"), ir.OpPrintChars, []byte("00"), nil)
	if err := e.ExecuteInstruction(inst); err == nil {
		t.Error("a broken pipe was taken for a successful print")
	}
}

// A host that wires no printer asked for a reading nobody can give. Printing nowhere by
// default would hide the wiring mistake in the one place it shows.
func TestAPrintWithNoPrinterIsAnError(t *testing.T) {
	e := New(NewEvaluatorOptions{})
	e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FromUint64(65))

	inst := ir.NewInstruction([]byte("02"), ir.OpPrintBytes, []byte("00"), nil)
	if err := e.ExecuteInstruction(inst); err == nil {
		t.Error("printing with no printer was taken for a print")
	}
}

// An assertion only runs under a runner that asked for it, and the opcode is what carries it
// there — a plain run consumes the operands and records nothing.
func TestExecuteInstructionDispatchesAnAssert(t *testing.T) {
	e := New(NewEvaluatorOptions{Asserts: true})
	e.environ.SetTemp(byteutil.ToHex([]byte("00")), byteutil.FalseTape(tapeSize))

	inst := ir.NewInstruction([]byte("02"), ir.OpAssert, []byte("00"), []byte("one is one"))
	if err := e.ExecuteInstruction(inst); err != nil {
		t.Fatalf("executing: %v", err)
	}

	results := e.GetAssertResults()
	if len(results) != 1 {
		t.Fatalf("recorded %d assertions, want one", len(results))
	}
	if results[0].Passed {
		t.Error("the assertion passed on a condition that does not hold")
	}
}

// An error raised by an operation comes back out of the dispatch rather than being swallowed
// on the way. Both of these also prove the opcode reached the operation it names: nothing
// else answers with these words.
func TestExecuteInstructionCarriesAnErrorBack(t *testing.T) {
	cases := []struct {
		name string
		inst ir.Instruction
		want string
	}{
		{
			name: "load of an identifier that was never set",
			inst: ir.NewInstruction([]byte("02"), ir.OpLoad, []byte("nope"), nil),
			want: "identifier nope not found",
		},
		{
			name: "call of an identifier that was never set",
			inst: ir.NewInstruction([]byte("02"), ir.OpCall, []byte("nope"), nil),
			want: "call: nope identifier not found",
		},
		{
			name: "divide by zero",
			inst: ir.NewInstruction([]byte("02"), ir.OpDivide, []byte("00"), []byte("01")),
			want: "integer divide by zero",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(NewEvaluatorOptions{})
			operands(6, 0)(e)

			err := e.ExecuteInstruction(tc.inst)
			if err == nil {
				t.Fatal("no error came back")
			}
			if err.Error() != tc.want {
				t.Errorf("answered %q, want %q", err, tc.want)
			}
		})
	}
}

// An opcode with no operation behind it steps over and says nothing. OpPreCall is the one
// declared and never emitted, which is exactly the shape of an opcode added tomorrow and not
// yet wired: a running program does not stop because of it.
func TestExecuteInstructionStepsOverAnOpcodeItDoesNotKnow(t *testing.T) {
	e := New(NewEvaluatorOptions{})

	inst := ir.NewInstruction([]byte("02"), ir.OpPreCall, []byte("00"), []byte("01"))
	if err := e.ExecuteInstruction(inst); err != nil {
		t.Fatalf("executing: %v", err)
	}

	if cursor, _ := e.GetInstructionsOffset(); cursor != 1 {
		t.Errorf("cursor landed on %d, want 1", cursor)
	}
	if got := e.environ.GetTemp(byteutil.ToHex([]byte("02"))); got != nil {
		t.Errorf("wrote %v under the label, want nothing", got)
	}
}

// Opcodes proven by a test of their own rather than by the table.
var coveredElsewhere = map[byte]bool{
	ir.OpPrintBytes:   true,
	ir.OpPrintChars:   true,
	ir.OpPrintDecimal: true,
	ir.OpAssert:       true,
	ir.OpCall:         true,
	// Declared and never emitted; the step-over test is what covers it.
	ir.OpPreCall: true,
}

// A new opcode is added to the emitter and wired into the evaluator in two separate places,
// and nothing connects them. This is what notices when the second half is missing.
func TestEveryOpcodeIsDispatchedByACase(t *testing.T) {
	tested := make(map[byte]bool, len(dispatchCases))
	for _, tc := range dispatchCases {
		tested[tc.opcode] = true
	}

	for op := ir.OpMultiply; op <= ir.OpField; op++ {
		if tested[op] || coveredElsewhere[op] {
			continue
		}
		t.Errorf("%s reaches no test through ExecuteInstruction", ir.ResolveOpCode(op))
	}
}
