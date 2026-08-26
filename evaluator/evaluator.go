package evaluator

import (
	"errors"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
	"github.com/guiferpa/aurora/wire/eval"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/module"
)

// A Printer is how a value leaves a program.
//
// The evaluator does not know what printing is — whether it reaches a terminal, a page or
// nothing at all — and it does not choose. It hands over the tape and keeps what comes back,
// which is the value the expression returns.
type Printer interface {
	Print(value []byte) ([]byte, error)
}

type Evaluator struct {
	// blocks is the program as blocks, when it was given as blocks. It is what a call reaches
	// into: a name bound to a scope holds the number of a block, and running the scope is
	// running from there.
	blocks        []ir.Block
	assertResults []eval.AssertResult // what each assertion did, in the order they ran
	asserts       bool                // whether assertions are evaluated at all
	printBytes    Printer
	printChars    Printer
	printDecimal  Printer
	environ       *environ.Environ
	// storage is what a chain keeps between transactions, kept here for one run of a program.
	// It is by the key as bytes, because a key is a tape and a tape is not a map key.
	storage  map[string][]byte
	tapeSize int
}

// TapeSize is the width, in bytes, of every value this evaluator handles.
func (e *Evaluator) TapeSize() int {
	return e.tapeSize
}

// ClearTemps drops the temps left behind by a program. A caller running two programs in
// one evaluator needs it: each is emitted on its own and numbers its labels from zero, so
// what one left behind would sit under the labels the next is about to use.
func (e *Evaluator) ClearTemps() {
	e.environ.ClearTemps()
}

// GetAssertResults returns every assertion that ran, in order.
func (e *Evaluator) GetAssertResults() []eval.AssertResult {
	return e.assertResults
}

// GetAssertErrors returns only the failures, which is what a plain run reports.
func (e *Evaluator) GetAssertErrors() []error {
	errs := make([]error, 0)
	for _, result := range e.assertResults {
		if !result.Passed {
			errs = append(errs, errors.New(result.Message))
		}
	}
	return errs
}

// value answers what an operand is worth.
//
// A Ref names a value another instruction left behind, so it is looked up. An Imm is the
// value: the program wrote it down and there is nothing to look up. Everything a position
// holding a value can be, and the only place that has to know it.
func (e *Evaluator) value(operand ir.Operand) []byte {
	if operand.Kind() == ir.KindImm {
		return operand.Bytes()
	}
	// A scope is worth the number of its block, and it is a value like any other — so it is a
	// tape of this program's width, not of whatever width the number was written in.
	if operand.Kind() == ir.KindBlock {
		return byteutil.PaddingTape(operand.Bytes(), e.tapeSize)
	}
	return e.environ.GetTemp(byteutil.ToHex(operand.Bytes()))
}

// operands reads the two values an operation consumes, as tapes of the configured size.
func (e *Evaluator) operands(left, right ir.Operand) (*uint256.Int, *uint256.Int) {
	x := byteutil.ToUint256(e.value(left), e.tapeSize)
	y := byteutil.ToUint256(e.value(right), e.tapeSize)
	return x, y
}

// setValue stores an arithmetic result, wrapped to the tape size: a tape of N bytes holds
// values modulo 2^(8N), so 255 + 1 is 0 when a tape is one byte wide.
func (e *Evaluator) setValue(label []byte, v *uint256.Int) {
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.FromUint256(v, e.tapeSize))
}

// setCondition stores a comparison result as an ordinary tape.
func (e *Evaluator) setCondition(label []byte, cond bool) {
	value := byteutil.FalseTape(e.tapeSize)
	if cond {
		value = byteutil.TrueTape(e.tapeSize)
	}
	e.environ.SetTemp(byteutil.ToHex(label), value)
}

func (e *Evaluator) EvaluateAdd(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Add(x, y))
	return nil
}

func (e *Evaluator) EvaluateSubtract(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Sub(x, y))
	return nil
}

func (e *Evaluator) EvaluateMultiply(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Mul(x, y))
	return nil
}

func (e *Evaluator) EvaluateDivide(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	if y.IsZero() {
		return fmt.Errorf("integer divide by zero")
	}
	e.setValue(label, new(uint256.Int).Div(x, y))
	return nil
}

func (e *Evaluator) EvaluateExponential(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Exp(x, y))
	return nil
}

func (e *Evaluator) EvaluateDiff(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, !x.Eq(y))
	return nil
}

func (e *Evaluator) EvaluateEquals(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Eq(y))
	return nil
}

func (e *Evaluator) EvaluateBigger(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Gt(y))
	return nil
}

func (e *Evaluator) EvaluateSmaller(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Lt(y))
	return nil
}

func (e *Evaluator) EvaluateAnd(label []byte, left, right ir.Operand) error {
	x := byteutil.ToBoolean(e.value(left))
	y := byteutil.ToBoolean(e.value(right))
	e.setCondition(label, x && y)
	return nil
}

func (e *Evaluator) EvaluateOr(label []byte, left, right ir.Operand) error {
	x := byteutil.ToBoolean(e.value(left))
	y := byteutil.ToBoolean(e.value(right))
	e.setCondition(label, x || y)
	return nil
}

// A tape is a shift register of fixed width. Pull shifts it left, letting the item in from
// the right; push shifts it right, letting the item in from the left. Whatever reaches the
// far end is discarded — the width never changes.
//
// The item contributes only its significant bytes (from its first non-zero byte on), so
// pulling 4 into a tape moves it one byte, not a whole tape width.

// EvaluatePush shifts the tape right and lets the item in at the left end.
func (e *Evaluator) EvaluatePush(label []byte, left, right ir.Operand) error {
	tape := byteutil.PaddingTape(e.value(left), e.tapeSize)
	item := byteutil.ExtractSignificantBytes(e.value(right))

	shifted := make([]byte, 0, len(tape)+len(item))
	shifted = append(shifted, item...)
	shifted = append(shifted, tape...)

	// Keeping the first bytes drops what was shifted out on the right.
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.LeadingTape(shifted, e.tapeSize))
	return nil
}

// EvaluateHead keeps the first n significant bytes of the tape.
//
// The index is taken modulo the tape width, so it can never be out of bounds: with
// one-byte tapes every index is 0.
// EvaluateJoin lays one tape after a run of them, which is how a shape is built. Each part
// is normalised to a tape first, so a run always holds a whole number of them however wide
// the value handed over was.
// EvaluateJoinOver lays one tape per field, end to end, over as many fields as the shape has.
//
// Every field crosses the same narrowing, which is what makes each one a single tape: a reel
// handed to a field would otherwise stay whole and the shape would come out longer than it
// declared.
func (e *Evaluator) EvaluateJoinOver(label []byte, operands []ir.Operand) error {
	joined := make([]byte, 0, len(operands)*e.tapeSize)
	for _, operand := range operands {
		joined = append(joined, byteutil.PaddingTape(e.value(operand), e.tapeSize)...)
	}

	e.environ.SetTemp(byteutil.ToHex(label), joined)
	return nil
}

// EvaluatePullOver shifts a tape left once per item, each entering at the right.
//
// The first operand is the tape the items are pulled onto, which for a literal is zeros.
// Keeping the last bytes after every item is what drops whatever was shifted off the left.
func (e *Evaluator) EvaluatePullOver(label []byte, operands []ir.Operand) error {
	tape := byteutil.PaddingTape(e.value(operands[0]), e.tapeSize)
	for _, operand := range operands[1:] {
		item := byteutil.ExtractSignificantBytes(e.value(operand))
		shifted := make([]byte, 0, len(tape)+len(item))
		shifted = append(shifted, tape...)
		shifted = append(shifted, item...)
		tape = byteutil.PaddingTape(shifted, e.tapeSize)
	}

	e.environ.SetTemp(byteutil.ToHex(label), tape)
	return nil
}

// EvaluateField takes one tape out of a run, by index. The index is a literal operand
// written inline by the emitter, resolved from a shape declaration that no longer exists.
//
// Reading past the end gives the neutral value rather than failing, the same answer feed
// gives past the end of what was applied: an operation on tapes does not stop a running
// program.
func (e *Evaluator) EvaluateField(label []byte, left, right ir.Operand) error {
	run := e.value(left)
	start := int(byteutil.ToUint64(right.Bytes())) * e.tapeSize

	value := byteutil.FalseTape(e.tapeSize)
	if start >= 0 && start+e.tapeSize <= len(run) {
		value = run[start : start+e.tapeSize]
	}

	e.environ.SetTemp(byteutil.ToHex(label), value)
	return nil
}

func (e *Evaluator) EvaluateHead(label []byte, left, right ir.Operand) error {
	significant, n := e.tapeSlice(left, right)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.PaddingTape(significant[:n], e.tapeSize))
	return nil
}

// EvaluateTail drops the first n significant bytes of the tape and keeps the rest.
func (e *Evaluator) EvaluateTail(label []byte, left, right ir.Operand) error {
	significant, n := e.tapeSlice(left, right)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.PaddingTape(significant[n:], e.tapeSize))
	return nil
}

// tapeSlice reads the tape under left and the index under right, and returns the tape's
// significant bytes with the index already normalized into them.
func (e *Evaluator) tapeSlice(left, right ir.Operand) ([]byte, int) {
	tape := byteutil.PaddingTape(e.value(left), e.tapeSize)
	significant := byteutil.ExtractSignificantBytes(tape)

	// The index is what the operation takes about itself, not a value: it is written inline.
	n := int(byteutil.ToUint64(right.Bytes()) % uint64(e.tapeSize))
	if n > len(significant) {
		n = len(significant)
	}
	return significant, n
}

// The three print builtins read the same tape three ways, and the evaluator knows none of the
// three: it hands the value to whoever was given for that reading and keeps what came back.
//
// What comes back is the value of the expression. Everything in Aurora returns
// something, and a print used to be the exception — it left nothing under its label, which is
// why a line of "printd 42" in the REPL showed no value at all.
func (e *Evaluator) EvaluatePrintBytes(label []byte, left ir.Operand) error {
	return e.print(e.printBytes, label, left)
}

func (e *Evaluator) EvaluatePrintChars(label []byte, left ir.Operand) error {
	return e.print(e.printChars, label, left)
}

func (e *Evaluator) EvaluatePrintDecimal(label []byte, left ir.Operand) error {
	return e.print(e.printDecimal, label, left)
}

// print reads the value, hands it to the port, and stores what the port answered.
//
// The error is returned rather than dropped: writing used to be "_, _ =", so a program whose
// output went nowhere — a closed pipe — carried on as if it had been heard.
func (e *Evaluator) print(printer Printer, label []byte, left ir.Operand) error {
	val := e.value(left)
	if printer == nil {
		return fmt.Errorf("no printer was given for this reading")
	}

	printed, err := printer.Print(val)
	if err != nil {
		return err
	}

	e.environ.SetTemp(byteutil.ToHex(label), printed)
	return nil
}

func (e *Evaluator) EvaluateSave(label []byte, left, right ir.Operand) error {
	e.environ.SetTemp(byteutil.ToHex(label), e.value(left))
	return nil
}

// resolve answers what a name is bound to, and the environ it lives in.
//
// Two hops. The chain first — every scope open around here, which is what a deferred scope
// sees of whoever called it, and what has always been the whole of a lookup. Then the environ
// of the module the name says it belongs to, which is why a name inside a module carries the
// module in front of it: a scope from another file, running in somebody else's chain, still
// finds what its own file bound.
//
// The environ comes back with the value because a deferred scope is an index counted in the
// environ that created it. A call has to look for the body where it found the name, and a
// second module's index 0 is a different scope from the first one's.
func (e *Evaluator) resolve(name []byte) ([]byte, *environ.Environ) {
	key := byteutil.ToHex(name)
	if home := e.environ.Holder(key); home != nil {
		return home.GetLocalIdent(key), home
	}
	id, _, qualified := module.Split(string(name))
	if !qualified {
		return nil, nil
	}
	home := e.environ.Module(string(id))
	if home == nil {
		return nil, nil
	}
	return home.GetLocalIdent(key), home
}

func (e *Evaluator) EvaluateLoad(label []byte, left, right ir.Operand) error {
	val, _ := e.resolve(left.Bytes())
	if val == nil {
		return fmt.Errorf("identifier %s not found", left.Bytes())
	}
	e.environ.SetTemp(byteutil.ToHex(label), val)
	return nil
}

func (e *Evaluator) EvaluateIdent(label []byte, left, right ir.Operand) error {
	k := byteutil.ToHex(left.Bytes())
	if v := e.environ.GetLocalIdent(k); v != nil {
		return fmt.Errorf("conflict between identifiers named %s", left.Bytes())
	}
	val := e.value(right)
	e.environ.SetIdent(k, val)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.FalseTape(e.tapeSize))
	return nil
}

func (e *Evaluator) EvaluateGetArg(label []byte, left, right ir.Operand) error {
	index := byteutil.ToUint64(left.Bytes())
	v := builtin.FeedFunction(e.environ.GetArguments(), index, e.tapeSize)
	l := byteutil.ToHex(label)
	e.environ.SetTemp(l, v)
	return nil
}

// EvaluateCall runs the scope a name reaches, over the values the call carries.
//
// The values arrive as operands, so the environ built for the call holds exactly them. They
// used to arrive through the environ of whoever was calling, written there by an instruction
// each — which meant a scope calling another handed over its own arguments as well, wherever
// the call applied fewer values than the caller had received.
func (e *Evaluator) EvaluateCallOver(label []byte, operands []ir.Operand) error {
	left := operands[0].Bytes()
	val, _ := e.resolve(left)
	if val == nil {
		return fmt.Errorf("call: %s identifier not found", left)
	}
	index := byteutil.ToUint256(val, e.tapeSize).Uint64()

	// A name bound to a scope holds the number of a block, and running the scope is running
	// from there. A value that names no block is a value with no scope behind it — a number,
	// or what a block returned — and block zero is the top of the program, which nothing
	// is bound to.
	if int(index) >= len(e.blocks) || index == 0 {
		return fmt.Errorf("call: value is not a deferred scope")
	}

	// The values arrive as operands, so the place built for the call holds exactly them. They
	// used to arrive through the place of whoever was calling, written there an instruction
	// each — which meant a scope calling another handed over its own values as well, wherever
	// the call applied fewer than the caller had received.
	args := make(map[uint64][]byte, len(operands)-1)
	for at, operand := range operands[1:] {
		args[uint64(at)] = e.value(operand)
	}
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	next.SetArguments(args)

	outer := e.environ
	e.environ = outer.Ahead(next)
	answered, err := e.RunBlocks(e.blocks, ir.Point{Block: ir.BlockID(index)}, nil)
	e.environ = outer
	if err != nil {
		return err
	}
	if answered == nil {
		answered = byteutil.FalseTape(e.tapeSize)
	}
	e.environ.SetTemp(byteutil.ToHex(label), answered)
	return nil
}

// EvaluateAssert checks a condition, but only under a runner that asked for it. A plain
// run consumes the operands and moves on: assertions belong to "aurora test", and a
// program that happens to hold one should not fail because of it.
func (e *Evaluator) EvaluateAssert(label []byte, left, right ir.Operand) error {
	cond := e.value(left)
	// The message is written inline by the emitter, not held in a temp.
	msg := string(right.Bytes())

	if !e.asserts {
		e.environ.SetTemp(byteutil.ToHex(label), byteutil.FalseTape(e.tapeSize))
		return nil
	}

	passed, failure := builtin.AssertFunction(cond, msg)
	result := eval.AssertResult{Passed: passed}
	if passed {
		result.Message = msg
	} else {
		result.Message = failure.Error()
	}
	e.assertResults = append(e.assertResults, result)

	e.environ.SetTemp(byteutil.ToHex(label), byteutil.FalseTape(e.tapeSize))
	return nil
}

// operation is what an opcode runs: the label the result is written under, and the two
// operands the emitter wrote beside it.
type operation func(e *Evaluator, label []byte, left, right ir.Operand) error

// Which operation each opcode runs.
//
// This was a chain of thirty ifs, each asking the opcode again, so reaching an assertion
// meant answering twenty-nine questions that had nothing to do with it — and every new
// instruction in the language made the chain one question longer. The table answers in one
// step and, more to the point, an opcode and its operation now sit on the same line.
//
// It is filled in init rather than where it is declared because a call runs a body of
// instructions: EvaluateCall reaches ExecuteInstruction, which reads this table, and Go
// reads that as a variable initialising itself.
var operations map[byte]operation

func init() {
	operations = map[byte]operation{
		// Arithmetic
		ir.OpAdd:         (*Evaluator).EvaluateAdd,
		ir.OpSubtract:    (*Evaluator).EvaluateSubtract,
		ir.OpMultiply:    (*Evaluator).EvaluateMultiply,
		ir.OpDivide:      (*Evaluator).EvaluateDivide,
		ir.OpExponential: (*Evaluator).EvaluateExponential,

		// Comparison
		ir.OpDiff:    (*Evaluator).EvaluateDiff,
		ir.OpEquals:  (*Evaluator).EvaluateEquals,
		ir.OpBigger:  (*Evaluator).EvaluateBigger,
		ir.OpSmaller: (*Evaluator).EvaluateSmaller,

		// Logic
		ir.OpAnd: (*Evaluator).EvaluateAnd,
		ir.OpOr:  (*Evaluator).EvaluateOr,

		// Builtins. A print reads one operand, and the opcode is the whole difference between
		// the three readings of it.
		ir.OpPrintBytes: func(e *Evaluator, label []byte, left, _ ir.Operand) error {
			return e.EvaluatePrintBytes(label, left)
		},
		ir.OpPrintChars: func(e *Evaluator, label []byte, left, _ ir.Operand) error {
			return e.EvaluatePrintChars(label, left)
		},
		ir.OpPrintDecimal: func(e *Evaluator, label []byte, left, _ ir.Operand) error {
			return e.EvaluatePrintDecimal(label, left)
		},

		// Memory
		ir.OpSave:  (*Evaluator).EvaluateSave,
		ir.OpLoad:  (*Evaluator).EvaluateLoad,
		ir.OpIdent: (*Evaluator).EvaluateIdent,

		// Control is not an instruction. Where a program goes next is what a block's
		// terminator says, and an instruction computes a value and does nothing else — which
		// is why there is nothing to run for OpIf, OpJump, OpBeginScope or OpReturn. They are
		// how the emitter writes structure down on its way to the blocks, and the crossing
		// consumes them.

		// Arguments
		ir.OpGetFeed: (*Evaluator).EvaluateGetArg,

		// Tape operations
		ir.OpPush:  (*Evaluator).EvaluatePush,
		ir.OpField: (*Evaluator).EvaluateField,
		ir.OpHead:  (*Evaluator).EvaluateHead,
		ir.OpTail:  (*Evaluator).EvaluateTail,

		// Storage
		ir.OpStorageGet: (*Evaluator).EvaluateStorageGet,
		ir.OpStorageSet: (*Evaluator).EvaluateStorageSet,

		// Assertions
		ir.OpAssert: (*Evaluator).EvaluateAssert,
	}
}

// runOperations are the instructions that take as many operands as they were given, rather
// than a pair. A construction is the case for it: a shape has as many fields as it has, and a
// tape literal as many items.
//
// They are a map of their own rather than a special case inside the dispatch, because two
// shapes of instruction are two shapes of operation — the same reason the IR has
// NewInstruction beside NewInstructionOver.
var runOperations map[byte]func(*Evaluator, []byte, []ir.Operand) error

func init() {
	// Built here rather than declared, for the same reason the pair of them is: a call runs
	// instructions, and running instructions reads this map.
	runOperations = map[byte]func(*Evaluator, []byte, []ir.Operand) error{
		ir.OpJoin: (*Evaluator).EvaluateJoinOver,
		ir.OpPull: (*Evaluator).EvaluatePullOver,
		ir.OpCall: (*Evaluator).EvaluateCallOver,
	}
}

// RunBlocks runs from a block until it answers, or until control arrives somewhere the caller
// wanted it to stop.
//
// A block is a run of instructions with one way in and one way out, so running one is running
// its instructions in order and then reading its terminator. Nothing inside can send control
// anywhere, which is why there is no cursor here: the old walk moved one because an "if"
// carried how many instructions to skip, and a block names where it goes instead.
//
// Stopping is a place rather than a count for the same reason. A program made of several files
// runs each of them with its own names, and where one ends is where the next begins — which is
// a block, not an offset.
func (e *Evaluator) RunBlocks(blocks []ir.Block, from ir.Point, until func(ir.Point) bool) ([]byte, error) {
	held := e.blocks
	e.blocks = blocks
	defer func() { e.blocks = held }()

	id, at := from.Block, from.At
	for {
		// Arriving somewhere is the last thing the run before it does — the value it worked
		// out is handed over on the way in — so the stop is read here, once control is
		// already there and the handover is done. Reading it before would leave the value of
		// whatever was running behind.
		if until != nil && until(ir.Point{Block: id, At: at}) {
			return nil, nil
		}
		if int(id) >= len(blocks) {
			return nil, fmt.Errorf("no block numbered %d", id)
		}
		block := blocks[id]
		for ; at < len(block.Insts); at++ {
			// Stopping is checked in front of each instruction rather than only on the way
			// into a block, because where a run is meant to stop is where something else
			// begins, and that can be halfway through one.
			if until != nil && until(ir.Point{Block: id, At: at}) {
				return nil, nil
			}
			if err := e.ExecuteInstruction(block.Insts[at]); err != nil {
				return nil, err
			}
		}

		term := block.Term
		if term.Kind == ir.Ret {
			return e.returned(term.Value), nil
		}

		target := term.Targets[0]
		if term.Kind == ir.BrIf && !byteutil.ToBoolean(e.value(term.Cond)) {
			target = term.Targets[1]
		}
		id, at = e.arrive(blocks[target.Block], target), 0
	}
}

// answered reads what a terminator carries, and answers the neutral tape where there is
// nothing to read. An empty block still has a value, and so does a scope whose last expression
// bound a name rather than computing one.
func (e *Evaluator) returned(operand ir.Operand) []byte {
	if value := e.value(operand); value != nil {
		return value
	}
	return byteutil.FalseTape(e.tapeSize)
}

// arrive goes to a block, opening or closing a place for names on the way.
//
// Arriving with values closes one. A branch hands the value of the arm that ran to where the
// arms meet, and a block written inside an expression hands its value to where the run carries
// on — and in both, what was bound inside is done with. The values are read before the place
// closes and written down after, which is what lets the name a value is known by afterwards be
// set in the place that will read it.
//
// Arriving with none opens one. That is going into an arm, or into a block written inside an
// expression: what it binds is its own, so the "x" of an inner block is not the "x" of the one
// around it. The values applied to the running scope come in with it, because a block is not
// applied anything of its own and feed still means the enclosing vector.
//
// A scope's parameters are unnamed and nothing is written down for them: they arrive as the
// vector applied to it, which whoever called it put in place.
func (e *Evaluator) arrive(block ir.Block, target ir.Target) ir.BlockID {
	if len(target.Args) == 0 {
		next := environ.NewEnviron(environ.NewEnvironOptions{})
		next.SetArguments(e.environ.GetArguments())
		e.environ = e.environ.Ahead(next)
		return target.Block
	}

	values := make([][]byte, 0, len(target.Args))
	for _, arg := range target.Args {
		values = append(values, e.returned(arg))
	}

	e.environ = e.environ.GetPrevious()
	for at, value := range values {
		if at >= len(block.Params) || len(block.Params[at]) == 0 {
			continue
		}
		e.environ.SetTemp(byteutil.ToHex(block.Params[at]), value)
	}
	return target.Block
}

// EvaluateBlocks runs a program given as blocks, from a block, with names of its own when it
// is a file of a program made of several.
//
// It is the way in for a runner that has a whole program in hand: aurora run and aurora test.
// Where one file ends is where the next begins, so what stops it is arriving at that block.
func (e *Evaluator) EvaluateBlocks(blocks []ir.Block, from ir.Point, until func(ir.Point) bool, id string) (eval.Returns, error) {
	if id == "" {
		if _, err := e.RunBlocks(blocks, from, until); err != nil {
			return nil, err
		}
		return e.environ.GetTemps(), nil
	}

	outer := e.environ
	e.environ = outer.OpenModule(id)
	defer func() { e.environ = outer }()

	if _, err := e.RunBlocks(blocks, from, until); err != nil {
		return nil, err
	}
	return e.environ.GetTemps(), nil
}

// ExecuteInstruction runs one instruction: the opcode names the operation, and the operation
// moves the cursor itself, since where it lands is part of what the instruction does.
//
// An opcode with no operation behind it is stepped over rather than refused. Every opcode the
// IR declares has one today, so what this is for is the next one: an opcode added to the IR
// and wired into one consumer before the other is exactly what a half-wired new opcode looks
// like, and a running program does not stop for it.
func (e *Evaluator) ExecuteInstruction(inst ir.Instruction) error {
	if over, ok := runOperations[inst.GetOpCode()]; ok {
		return over(e, inst.GetLabel(), inst.GetOperands())
	}
	op, ok := operations[inst.GetOpCode()]
	if !ok {
		return nil
	}
	return op(e, inst.GetLabel(), inst.GetLeft(), inst.GetRight())
}

type NewEvaluatorOptions struct {
	// A Printer per reading of a tape. What each one does with the value, and what it
	// returns, is the host's business: the evaluator only asks and keeps the answer.
	PrintBytes   Printer
	PrintChars   Printer
	PrintDecimal Printer
	Args         []byte
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
	// Asserts turns assertions on. Only "aurora test" does.
	Asserts bool
}

func New(options NewEvaluatorOptions) *Evaluator {
	return &Evaluator{
		assertResults: make([]eval.AssertResult, 0),
		asserts:       options.Asserts,
		storage:       make(map[string][]byte),
		printBytes:    options.PrintBytes,
		printChars:    options.PrintChars,
		printDecimal:  options.PrintDecimal,
		tapeSize:      byteutil.TapeSize(options.TapeSize),
		environ: environ.NewEnviron(environ.NewEnvironOptions{
			Args:     options.Args,
			TapeSize: byteutil.TapeSize(options.TapeSize),
		}),
	}
}
