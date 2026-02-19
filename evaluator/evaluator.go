package evaluator

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
)

// encodeDeferBlob serializes a deferred scope into a blob for storage in environ.defers.
// Layout: [0:8] from (uint64 BE), [8:16] to (uint64 BE), [16] keyLen, [17:17+N] returnKey.
// Total length: 17 + len(returnKey).
func encodeDeferBlob(from, to uint64, returnKey string) []byte {
	key := []byte(returnKey)
	b := make([]byte, 0, 17+len(key))
	b = append(b, byteutil.FromUint64(from)...)
	b = append(b, byteutil.FromUint64(to)...)
	b = append(b, byte(len(key)))
	b = append(b, key...)
	return b
}

// decodeDeferBlob parses a blob from encodeDeferBlob.
// Returns (from, to, returnKey, true) or (0, 0, "", false) if val is too short or invalid.
func decodeDeferBlob(val []byte) (from, to uint64, returnKey string, ok bool) {
	const minLen = 17
	if len(val) < minLen {
		return 0, 0, "", false
	}
	from = binary.BigEndian.Uint64(val[0:8])
	to = binary.BigEndian.Uint64(val[8:16])
	keyLen := int(val[16])
	if 17+keyLen > len(val) {
		return 0, 0, "", false
	}
	return from, to, string(val[17 : 17+keyLen]), true
}

type ReturnsPerLabel map[string][]byte

type Evaluator struct {
	player       *Player
	cursor       uint64
	end          uint64
	logger       *Logger
	insts        []emitter.Instruction
	assertErrors []error // Buffer to collect assert errors
	echoWriter   io.Writer
	printWriter  io.Writer
	environ      *environ.Environ
}

func (e *Evaluator) GetAssertErrors() []error {
	return e.assertErrors
}

func (e *Evaluator) SetPlayer(player *Player) {
	e.player = player
}

func (e *Evaluator) EvaluateAdd(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, x+y)
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSubtract(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, x-y)
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateMultiply(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, x*y)
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDivide(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	if y == 0 {
		return fmt.Errorf("integer divide by zero")
	}
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, x/y)
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateExponential(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	v := math.Pow(float64(x), float64(y))
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, uint64(v))
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDiff(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x != y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateEquals(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x == y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateBigger(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x > y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSmaller(label, left, right []byte) error {
	x := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToUint64(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x < y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateAnd(label, left, right []byte) error {
	x := byteutil.ToBoolean(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToBoolean(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x && y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateOr(label, left, right []byte) error {
	x := byteutil.ToBoolean(e.environ.GetTemp(byteutil.ToHex(left)))
	y := byteutil.ToBoolean(e.environ.GetTemp(byteutil.ToHex(right)))
	r := byteutil.False
	if x || y {
		r = byteutil.True
	}
	e.environ.SetTemp(byteutil.ToHex(label), r)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluatePrint(label, left []byte) error {
	val := e.environ.GetTemp(byteutil.ToHex(left))
	builtin.PrintFunction(e.printWriter, val)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateEcho(label, left []byte) error {
	val := e.environ.GetTemp(byteutil.ToHex(left))
	builtin.EchoFunction(e.echoWriter, val)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateSave(label, left, right []byte) error {
	e.environ.SetTemp(byteutil.ToHex(label), left)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateLoad(label, left, right []byte) error {
	val := e.environ.GetIdent(byteutil.ToHex(left))
	if val == nil {
		return fmt.Errorf("identifier %s not found", left)
	}
	e.environ.SetTemp(byteutil.ToHex(label), val)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateIf(label, left, right []byte) error {
	test := byteutil.ToBoolean(e.environ.GetTemp(byteutil.ToHex(left)))
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	next.SetArguments(e.environ.GetArguments())
	e.environ = e.environ.Ahead(next)
	if test {
		e.cursor++
		return nil
	}
	e.AddCursor(byteutil.ToUint64(right) + 1)
	return nil
}

func (e *Evaluator) EvaluateJump(label, left, right []byte) error {
	e.AddCursor(byteutil.ToUint64(left) + 1)
	return nil
}

func (e *Evaluator) EvaluateBeginScope(label, left, right []byte) error {
	next := environ.NewEnviron(environ.NewEnvironOptions{})
	e.environ = e.environ.Ahead(next)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateReturn(_, left, right []byte) error {
	label := byteutil.ToHex(left)
	value := e.environ.GetTemp(byteutil.ToHex(right))
	if value == nil {
		value = byteutil.Nothing
	}
	e.environ = e.environ.GetPrevious()
	e.environ.SetTemp(label, value)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateIdent(label, left, right []byte) error {
	k := byteutil.ToHex(left)
	if v := e.environ.GetLocalIdent(k); v != nil {
		return fmt.Errorf("conflict between identifiers named %s", left)
	}
	val := e.environ.GetTemp(byteutil.ToHex(right))
	e.environ.SetIdent(k, val)
	e.environ.SetTemp(byteutil.ToHex(label), byteutil.Nothing)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluatePushArg(label, left, right []byte) error {
	index := byteutil.ToUint64(left)
	val := e.environ.GetTemp(byteutil.ToHex(right))
	e.environ.SetArgument(index, val)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateGetArg(label, left, right []byte) error {
	index := byteutil.ToUint64(left)
	v := builtin.ArgumentsFunction(e.environ.GetArguments(), index)
	l := byteutil.ToHex(label)
	e.environ.SetTemp(l, v)
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) EvaluateDefer(label, left, right []byte) error {
	bodylength := byteutil.ToUint64(right)
	// e.cursor is the index of this OpDefer; the next instruction is the start of the deferred block (OpBeginScope).
	from := e.cursor + 1
	to := from + bodylength // index of OpReturn (last instruction of the block)
	returnKey := byteutil.ToHex(left)
	blob := encodeDeferBlob(from, to, returnKey)
	ref := byteutil.ToHex(byteutil.FromUint64(uint64(e.environ.DefersLength())))
	e.environ.SetDefer(ref, blob)
	e.environ.SetTemp(byteutil.ToHex(label), []byte(ref))
	e.AddCursor(1 + bodylength)
	return nil
}

func (e *Evaluator) EvaluateCall(label, left, right []byte) error {
	val := e.environ.GetIdent(byteutil.ToHex(left))
	if val == nil {
		return fmt.Errorf("call: identifier not found")
	}
	refKey := string(val)
	blob := e.environ.GetDefer(refKey)
	if blob == nil {
		return fmt.Errorf("call: value is not a deferred scope")
	}
	from, to, returnKey, ok := decodeDeferBlob(blob)
	if !ok {
		return fmt.Errorf("call: invalid deferred scope data")
	}
	args := e.environ.GetArguments()
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

func (e *Evaluator) EvaluateAssert(label, left, right []byte) error {
	cond := e.environ.GetTemp(byteutil.ToHex(left))
	msg := e.environ.GetTemp(byteutil.ToHex(right))
	passed, errMsg := builtin.AssertFunction(cond, msg)
	if !passed {
		e.assertErrors = append(e.assertErrors, errMsg)
	}
	e.IncrementCursor()
	return nil
}

func (e *Evaluator) CanReadInstructions() bool {
	return e.cursor < e.end
}

func (e *Evaluator) GetInstruction() emitter.Instruction {
	return e.insts[e.cursor]
}

func (e *Evaluator) SetInstructions(insts []emitter.Instruction) {
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

func (e *Evaluator) ExecuteInstruction(inst emitter.Instruction) error {
	// Arithmetic operations
	if inst.GetOpCode() == emitter.OpAdd {
		return e.EvaluateAdd(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpSubtract {
		return e.EvaluateSubtract(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpMultiply {
		return e.EvaluateMultiply(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpDivide {
		return e.EvaluateDivide(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpExponential {
		return e.EvaluateExponential(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Comparison operations
	if inst.GetOpCode() == emitter.OpDiff {
		return e.EvaluateDiff(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpEquals {
		return e.EvaluateEquals(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpBigger {
		return e.EvaluateBigger(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpSmaller {
		return e.EvaluateSmaller(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Logical operations
	if inst.GetOpCode() == emitter.OpAnd {
		return e.EvaluateAnd(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpOr {
		return e.EvaluateOr(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Builtins operations
	if inst.GetOpCode() == emitter.OpPrint {
		return e.EvaluatePrint(inst.GetLabel(), inst.GetLeft())
	}
	if inst.GetOpCode() == emitter.OpEcho {
		return e.EvaluateEcho(inst.GetLabel(), inst.GetLeft())
	}

	// Memory operations
	if inst.GetOpCode() == emitter.OpSave {
		return e.EvaluateSave(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpLoad {
		return e.EvaluateLoad(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpIdent {
		return e.EvaluateIdent(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Control flow operations
	if inst.GetOpCode() == emitter.OpIf {
		return e.EvaluateIf(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpJump {
		return e.EvaluateJump(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpBeginScope {
		return e.EvaluateBeginScope(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpReturn {
		return e.EvaluateReturn(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Arguments operations
	if inst.GetOpCode() == emitter.OpPushArg {
		return e.EvaluatePushArg(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpGetArg {
		return e.EvaluateGetArg(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Defer and call
	if inst.GetOpCode() == emitter.OpDefer {
		return e.EvaluateDefer(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}
	if inst.GetOpCode() == emitter.OpCall {
		return e.EvaluateCall(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	// Assertions
	if inst.GetOpCode() == emitter.OpAssert {
		return e.EvaluateAssert(inst.GetLabel(), inst.GetLeft(), inst.GetRight())
	}

	e.IncrementCursor()

	return nil
}

func (e *Evaluator) ExecuteInstructions(from, to uint64) (ReturnsPerLabel, error) {
	e.SetInstructionsOffset(from, to)

	for e.CanReadInstructions() {
		inst := e.GetInstruction()
		if err := e.logger.Println(inst); err != nil {
			return nil, err
		}
		if err := e.ExecuteInstruction(inst); err != nil {
			return nil, err
		}
	}

	return e.environ.GetTemps(), nil
}

func (e *Evaluator) Evaluate(insts []emitter.Instruction) (ReturnsPerLabel, error) {
	e.SetInstructions(insts)
	returns, err := e.ExecuteInstructions(0, uint64(len(e.insts)))
	if err := e.logger.Close(); err != nil {
		return nil, err
	}
	return returns, err
}

// EvaluateRange sets the full instruction slice and runs only the range [from, to).
// Used by the REPL: buffer accumulates all instructions; each line we append and run only the new slice,
// so defer from/to indices stay valid in the same buffer.
func (e *Evaluator) EvaluateRange(insts []emitter.Instruction, from, to uint64) (ReturnsPerLabel, error) {
	e.SetInstructions(insts)
	e.environ.ClearTemps()
	returns, err := e.ExecuteInstructions(from, to)
	if err := e.logger.Close(); err != nil {
		return nil, err
	}
	return returns, err
}

type NewEvaluatorOptions struct {
	EnableLogging bool
	Player        *Player
	EchoWriter    io.Writer
	PrintWriter   io.Writer
	Args          []byte
}

func New(options NewEvaluatorOptions) *Evaluator {
	return &Evaluator{
		player:       options.Player,
		cursor:       0,
		end:          0,
		logger:       NewLogger(options.EnableLogging),
		insts:        make([]emitter.Instruction, 0),
		assertErrors: make([]error, 0),
		echoWriter:   options.EchoWriter,
		printWriter:  options.PrintWriter,
		environ: environ.NewEnviron(environ.NewEnvironOptions{
			Args: options.Args,
		}),
	}
}
