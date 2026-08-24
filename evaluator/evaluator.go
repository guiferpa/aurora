package evaluator

import (
	"encoding/binary"
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

// deferMark opens every defer blob. The value handed to a program is an ordinary tape
// holding an index, so any number can be presented as a deferred scope; the mark is what
// keeps "call: value is not a deferred scope" an error rather than a silent jump into
// whatever scope happens to sit at that index.
const deferMark = 0xAE

// encodeDeferBlob serializes a deferred scope into a blob for storage in environ.defers.
// Layout: [0] mark, [1:9] from (uint64 BE), [9:17] to (uint64 BE), [17] keyLen,
// [18:18+N] returnKey. Total length: 18 + len(returnKey).
func encodeDeferBlob(from, to uint64, returnKey string) []byte {
	key := []byte(returnKey)
	b := make([]byte, 0, 18+len(key))
	b = append(b, deferMark)
	b = append(b, byteutil.FromUint64(from)...)
	b = append(b, byteutil.FromUint64(to)...)
	b = append(b, byte(len(key)))
	b = append(b, key...)
	return b
}

// decodeDeferBlob parses a blob from encodeDeferBlob.
// Returns (from, to, returnKey, true) or (0, 0, "", false) when val is too short, does not
// carry the mark, or is otherwise not a deferred scope.
func decodeDeferBlob(val []byte) (from, to uint64, returnKey string, ok bool) {
	const minLen = 18
	if len(val) < minLen || val[0] != deferMark {
		return 0, 0, "", false
	}
	from = binary.BigEndian.Uint64(val[1:9])
	to = binary.BigEndian.Uint64(val[9:17])
	keyLen := int(val[17])
	if 18+keyLen > len(val) {
		return 0, 0, "", false
	}
	return from, to, string(val[18 : 18+keyLen]), true
}

// deferKey is the internal key of a deferred scope, derived from its index. Keeping the key
// separate from the value lets the value be an ordinary tape, whatever the tape size.
func deferKey(index uint64) string {
	return byteutil.ToHex(byteutil.FromUint64(index))
}

// A Printer is how a value leaves a program.
//
// The evaluator does not know what printing is — whether it reaches a terminal, a page or
// nothing at all — and it does not choose. It hands over the tape and keeps what comes back,
// which is the value the expression answers with.
type Printer interface {
	Print(value []byte) ([]byte, error)
}

type Evaluator struct {
	// blocks is the program as blocks, when it was given as blocks. It is what a call reaches
	// into: a name bound to a scope holds the number of a block, and running the scope is
	// running from there.
	blocks        []ir.Block
	cursor        uint64
	end           uint64
	insts         []ir.Instruction
	assertResults []eval.AssertResult // what each assertion did, in the order they ran
	asserts       bool                // whether assertions are evaluated at all
	printBytes    Printer
	printChars    Printer
	printDecimal  Printer
	environ       *environ.Environ
	tapeSize      int
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
	// An immediate is the value, and a block is the number of a block — both are read off the
	// operand rather than looked up. Everything else names something another instruction left.
	if operand.Kind() == ir.KindImm || operand.Kind() == ir.KindBlock {
		return operand.Bytes()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSubtract(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Sub(x, y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateMultiply(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Mul(x, y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDivide(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	if y.IsZero() {
		return fmt.Errorf("integer divide by zero")
	}
	e.setValue(label, new(uint256.Int).Div(x, y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateExponential(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setValue(label, new(uint256.Int).Exp(x, y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDiff(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, !x.Eq(y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateEquals(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Eq(y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateBigger(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Gt(y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSmaller(label []byte, left, right ir.Operand) error {
	x, y := e.operands(left, right)
	e.setCondition(label, x.Lt(y))
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateAnd(label []byte, left, right ir.Operand) error {
	x := byteutil.ToBoolean(e.value(left))
	y := byteutil.ToBoolean(e.value(right))
	e.setCondition(label, x && y)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateOr(label []byte, left, right ir.Operand) error {
	x := byteutil.ToBoolean(e.value(left))
	y := byteutil.ToBoolean(e.value(right))
	e.setCondition(label, x || y)
	e.IncrementCursor()
	return nil
}

// A tape is a shift register of fixed width. Pull shifts it left, letting the item in from
// the right; push shifts it right, letting the item in from the left. Whatever reaches the
// far end is discarded — the width never changes.
//
// The item contributes only its significant bytes (from its first non-zero byte on), so
// pulling 4 into a tape moves it one byte, not a whole tape width.

// EvaluatePull shifts the tape left and lets the item in at the right end.
func (e *Evaluator) EvaluatePull(label []byte, left, right ir.Operand) error {
	tape := byteutil.PaddingTape(e.value(left), e.tapeSize)
	item := byteutil.ExtractSignificantBytes(e.value(right))

	shifted := make([]byte, 0, len(tape)+len(item))
	shifted = append(shifted, tape...)
	shifted = append(shifted, item...)

	// Keeping the last bytes drops what was shifted out on the left.
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.PaddingTape(shifted, e.tapeSize))
	e.IncrementCursor()
	return nil
}

// EvaluatePush shifts the tape right and lets the item in at the left end.
func (e *Evaluator) EvaluatePush(label []byte, left, right ir.Operand) error {
	tape := byteutil.PaddingTape(e.value(left), e.tapeSize)
	item := byteutil.ExtractSignificantBytes(e.value(right))

	shifted := make([]byte, 0, len(tape)+len(item))
	shifted = append(shifted, item...)
	shifted = append(shifted, tape...)

	// Keeping the first bytes drops what was shifted out on the right.
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.LeadingTape(shifted, e.tapeSize))
	e.IncrementCursor()
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
	e.IncrementCursor()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateJoin(label []byte, left, right ir.Operand) error {
	run := e.value(left)
	tape := byteutil.PaddingTape(e.value(right), e.tapeSize)

	joined := make([]byte, 0, len(run)+len(tape))
	joined = append(joined, run...)
	joined = append(joined, tape...)

	e.environ.SetTemp(byteutil.ToHex(label), joined)
	e.IncrementCursor()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateHead(label []byte, left, right ir.Operand) error {
	significant, n := e.tapeSlice(left, right)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.PaddingTape(significant[:n], e.tapeSize))
	e.IncrementCursor()
	return nil
}

// EvaluateTail drops the first n significant bytes of the tape and keeps the rest.
func (e *Evaluator) EvaluateTail(label []byte, left, right ir.Operand) error {
	significant, n := e.tapeSlice(left, right)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.PaddingTape(significant[n:], e.tapeSize))
	e.IncrementCursor()
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
// What comes back is the value of the expression. Everything in Aurora answers with
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSave(label []byte, left, right ir.Operand) error {
	e.environ.SetTemp(byteutil.ToHex(label), left.Bytes())
	e.IncrementCursor()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateIf(label []byte, left, right ir.Operand) error {
	test := byteutil.ToBoolean(e.value(left))
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	next.SetArguments(e.environ.GetArguments())
	e.environ = e.environ.Ahead(next)
	if test {
		e.cursor++
		return nil
	}
	e.AddCursor(byteutil.ToUint64(right.Bytes()) + 1)
	return nil
}

func (e *Evaluator) EvaluateJump(label []byte, left, right ir.Operand) error {
	e.AddCursor(byteutil.ToUint64(left.Bytes()) + 1)
	return nil
}

// EvaluateBeginScope opens a block: a place for names, inside whatever is running.
//
// It carries the values applied to the running scope in with it. A block is not applied
// anything of its own — nothing calls it, control walks into it — so "the vector applied to
// this scope" is still the enclosing one's, and feed reads it. An arm of an "if" already did
// this; a block did not, so a block written inside a scope could not read what the scope was
// called with, and answered as if it had been called with nothing.
func (e *Evaluator) EvaluateBeginScope(label []byte, left, right ir.Operand) error {
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	next.SetArguments(e.environ.GetArguments())
	e.environ = e.environ.Ahead(next)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateReturn(_ []byte, left, right ir.Operand) error {
	label := byteutil.ToHex(left.Bytes())
	value := e.value(right)
	if value == nil {
		value = byteutil.FalseTape(e.tapeSize)
	}
	e.environ = e.environ.GetPrevious()
	e.environ.SetTemp(label, value)
	e.IncrementCursor()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateGetArg(label []byte, left, right ir.Operand) error {
	index := byteutil.ToUint64(left.Bytes())
	v := builtin.FeedFunction(e.environ.GetArguments(), index, e.tapeSize)
	l := byteutil.ToHex(label)
	e.environ.SetTemp(l, v)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDefer(label []byte, left, right ir.Operand) error {
	bodylength := byteutil.ToUint64(right.Bytes())
	// e.cursor is the index of this OpDefer; the next instruction is the start of the deferred block (OpBeginScope).
	from := e.cursor + 1
	to := from + bodylength // index of OpReturn (last instruction of the block)
	returnKey := byteutil.ToHex(left.Bytes())

	// The value of a defer is its index as a tape, like every other value in the language.
	// It used to be the hex key itself — 16 bytes of ASCII text that ignored the tape size.
	index := uint64(e.environ.DefersLength())
	e.environ.SetDefer(deferKey(index), encodeDeferBlob(from, to, returnKey))
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.FromUint256(uint256.NewInt(index), e.tapeSize))

	e.AddCursor(1 + bodylength)
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
	val, home := e.resolve(left)
	if val == nil {
		return fmt.Errorf("call: %s identifier not found", left)
	}
	index := byteutil.ToUint256(val, e.tapeSize).Uint64()

	// A name bound to a scope holds the number of a block, and running the scope is running
	// from there. It used to hold an index into a list of scopes, each of which recorded where
	// its body sat as a stretch of the instructions being executed — so a call was a range,
	// and the value it answered with had to be fetched afterwards from a key both sides agreed
	// on. A block ends by answering, so the answer comes back from running it.
	if e.blocks != nil {
		args := make(map[uint64][]byte, len(operands)-1)
		for at, operand := range operands[1:] {
			args[uint64(at)] = e.value(operand)
		}
		next := environ.NewEnviron(environ.NewEnvironOptions{})
		next.SetArguments(args)

		outer := e.environ
		e.environ = outer.Ahead(next)
		answered, err := e.RunBlocks(e.blocks, ir.BlockID(index), nil)
		e.environ = outer
		if err != nil {
			return err
		}
		if answered == nil {
			answered = byteutil.FalseTape(e.tapeSize)
		}
		e.environ.SetTemp(byteutil.ToHex(label), answered)
		e.IncrementCursor()
		return nil
	}

	blob := home.GetLocalDefer(deferKey(index))
	if blob == nil {
		return fmt.Errorf("call: value is not a deferred scope")
	}
	from, to, returnKey, ok := decodeDeferBlob(blob)
	if !ok {
		return fmt.Errorf("call: invalid deferred scope data")
	}
	args := make(map[uint64][]byte, len(operands)-1)
	for at, operand := range operands[1:] {
		args[uint64(at)] = e.value(operand)
	}
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	next.SetArguments(args)
	e.environ = e.environ.Ahead(next)
	savedCursor, savedEnd := e.cursor, e.end
	_, err := e.ExecuteInstructions(from+1, to)
	e.cursor, e.end = savedCursor, savedEnd
	if err != nil {
		return err
	}
	retval := e.environ.GetTemp(returnKey)
	e.environ.SetTemp(byteutil.ToHex(label), retval)
	e.IncrementCursor()
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
		e.IncrementCursor()
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
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) CanReadInstructions() bool {
	return e.cursor < e.end
}

func (e *Evaluator) GetInstruction() ir.Instruction {
	return e.insts[e.cursor]
}

func (e *Evaluator) SetInstructions(insts []ir.Instruction) {
	e.insts = insts
}

func (e *Evaluator) SetInstructionsOffset(begin, end uint64) {
	e.cursor = begin
	e.end = end
}

func (e *Evaluator) GetInstructionsOffset() (uint64, uint64) {
	return e.cursor, e.end
}

func (e *Evaluator) IncrementCursor() {
	e.cursor++
}

func (e *Evaluator) AddCursor(offset uint64) {
	e.cursor += offset
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

		// Control flow
		ir.OpIf:         (*Evaluator).EvaluateIf,
		ir.OpJump:       (*Evaluator).EvaluateJump,
		ir.OpBeginScope: (*Evaluator).EvaluateBeginScope,
		ir.OpReturn:     (*Evaluator).EvaluateReturn,

		// Arguments
		ir.OpGetFeed: (*Evaluator).EvaluateGetArg,

		// Defer
		ir.OpDefer: (*Evaluator).EvaluateDefer,

		// Tape operations
		ir.OpPull:  (*Evaluator).EvaluatePull,
		ir.OpPush:  (*Evaluator).EvaluatePush,
		ir.OpJoin:  (*Evaluator).EvaluateJoin,
		ir.OpField: (*Evaluator).EvaluateField,
		ir.OpHead:  (*Evaluator).EvaluateHead,
		ir.OpTail:  (*Evaluator).EvaluateTail,

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
func (e *Evaluator) RunBlocks(blocks []ir.Block, from ir.BlockID, until func(ir.BlockID) bool) ([]byte, error) {
	held := e.blocks
	e.blocks = blocks
	defer func() { e.blocks = held }()

	id := from
	for {
		if int(id) >= len(blocks) {
			return nil, fmt.Errorf("no block numbered %d", id)
		}
		block := blocks[id]
		for _, inst := range block.Insts {
			if err := e.ExecuteInstruction(inst); err != nil {
				return nil, err
			}
		}

		term := block.Term
		if term.Kind == ir.Ret {
			return e.answered(term.Value), nil
		}

		target := term.Targets[0]
		if term.Kind == ir.BrIf && !byteutil.ToBoolean(e.value(term.Cond)) {
			target = term.Targets[1]
		}
		if until != nil && until(target.Block) {
			return nil, nil
		}
		id = e.arrive(blocks[target.Block], target)
	}
}

// answered reads what a terminator carries, and answers the neutral tape where there is
// nothing to read. An empty block still has a value, and so does a scope whose last expression
// bound a name rather than computing one.
func (e *Evaluator) answered(operand ir.Operand) []byte {
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
		values = append(values, e.answered(arg))
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
func (e *Evaluator) EvaluateBlocks(blocks []ir.Block, from ir.BlockID, until func(ir.BlockID) bool, id string) (eval.Returns, error) {
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
		e.IncrementCursor()
		return nil
	}
	return op(e, inst.GetLabel(), inst.GetLeft(), inst.GetRight())
}

func (e *Evaluator) ExecuteInstructions(from, to uint64) (eval.Returns, error) {
	e.SetInstructionsOffset(from, to)

	for e.CanReadInstructions() {
		inst := e.GetInstruction()
		if err := e.ExecuteInstruction(inst); err != nil {
			return nil, err
		}
	}

	return e.environ.GetTemps(), nil
}

func (e *Evaluator) Evaluate(insts []ir.Instruction) (eval.Returns, error) {
	e.SetInstructions(insts)
	returns, err := e.ExecuteInstructions(0, uint64(len(e.insts)))
	return returns, err
}

// EvaluateRange sets the full instruction slice and runs only the range [from, to).
// Used by the REPL: buffer accumulates all instructions; each line we append and run only the new slice,
// so defer from/to indices stay valid in the same buffer.
func (e *Evaluator) EvaluateRange(insts []ir.Instruction, from, to uint64) (eval.Returns, error) {
	e.SetInstructions(insts)
	e.environ.ClearTemps()
	returns, err := e.ExecuteInstructions(from, to)
	return returns, err
}

// EvaluateModule runs one module's range, in the environ that module's names belong to.
//
// A program of several files is one stream of instructions, and each module is a range of it
// — a call reaching across modules lands on a body that has to be there, so the stream is
// never sliced. What changes from range to range is where the names go: a module binds into
// an environ of its own, which is what lets two modules bind the same word.
//
// The empty name is the file somebody asked to run. It has no environ of its own; it has the
// one every chain ends at, which is where a program with no modules at all has always run.
func (e *Evaluator) EvaluateModule(insts []ir.Instruction, from, to uint64, id string) (eval.Returns, error) {
	if id == "" {
		return e.EvaluateRange(insts, from, to)
	}

	outer := e.environ
	e.environ = outer.OpenModule(id)
	defer func() { e.environ = outer }()

	return e.EvaluateRange(insts, from, to)
}

type NewEvaluatorOptions struct {
	// A Printer per reading of a tape. What each one does with the value, and what it
	// answers with, is the host's business: the evaluator only asks and keeps the answer.
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
		cursor:        0,
		end:           0,
		insts:         make([]ir.Instruction, 0),
		assertResults: make([]eval.AssertResult, 0),
		asserts:       options.Asserts,
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
