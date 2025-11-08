package evaluator

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
)

type Evaluator struct {
	player       *Player
	envpool      *environ.Pool
	cursor       uint64
	insts        []emitter.Instruction
	currseg      *environ.ScopeCallable
	result       [][]byte
	counter      *uint64
	debug        bool
	assertErrors []string // Buffer to collect assert errors
}

func isTemp(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[0] == 0x30 { // 0
		return true
	}
	return false
}

func (e *Evaluator) walkTemps(bs []byte) []byte {
	l := byteutil.ToHex(bs)
	bs = e.envpool.GetTemp(l)
	if isTemp(bs) {
		return e.walkTemps(bs)
	}
	return bs
}

// bytesToUint64ForArithmetic converts byte arrays to uint64 for arithmetic operations.
// Note: In Aurora's untyped design, all values are byte arrays. For arithmetic operations,
// we interpret the first 8 bytes as a uint64. If the array is larger than 8 bytes,
// only the first 8 bytes are used (right-aligned for arrays < 8 bytes via padding).
// This is a design decision: arithmetic operations work on 64-bit integers.
func bytesToUint64ForArithmetic(bs []byte) uint64 {
	padded := byteutil.Padding64Bits(bs)
	// If array is larger than 8 bytes, we only use the first 8 bytes
	// This means tapes larger than 8 bytes will have their extra bytes ignored
	if len(bs) > 8 {
		// Use first 8 bytes (most significant bytes in big-endian)
		return binary.BigEndian.Uint64(padded[:8])
	}
	return binary.BigEndian.Uint64(padded)
}

func (e *Evaluator) exec(label []byte, op byte, left, right []byte) error {
	// For OpAssert, left is a label reference, don't resolve it
	if op != emitter.OpAssert {
		if isTemp(left) {
			left = e.walkTemps(left)
		}
	}

	if isTemp(right) {
		right = e.walkTemps(right)
	}

	if op == emitter.OpSave {
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)
		e.envpool.SetTemp(l, left)
		e.cursor++
		return nil
	}

	if op == emitter.OpPull {
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)

		// Ensure right is exactly 8 bytes (handles both uint64 and direct bytes)
		rightDirect := byteutil.Padding64Bits(right)

		// Ensure left is exactly 8 bytes
		leftDirect := byteutil.Padding64Bits(left)

		// Extract significant bytes from right (bytes from first non-zero to end)
		rightSignificantBytes := byteutil.ExtractSignificantBytes(rightDirect)

		// Extract significant bytes from left (bytes from first non-zero to end)
		leftSignificantBytes := byteutil.ExtractSignificantBytes(leftDirect)

		// pull: remove bytes from beginning of left, add bytes from right at the end
		// Concatenate: left bytes + right bytes
		result := append(leftSignificantBytes, rightSignificantBytes...)

		// If result exceeds 8 bytes, keep only the last 8 bytes
		if len(result) > 8 {
			result = result[len(result)-8:]
		}

		// Pad to exactly 8 bytes with right-alignment
		result = byteutil.Padding64Bits(result)

		e.envpool.SetTemp(l, result)
		e.cursor++
		return nil
	}

	if op == emitter.OpPush {
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)

		// Ensure right is exactly 8 bytes (handles both uint64 and direct bytes)
		rightDirect := byteutil.Padding64Bits(right)

		// Ensure left is exactly 8 bytes
		leftDirect := byteutil.Padding64Bits(left)

		// Extract significant bytes from right (bytes from first non-zero to end)
		rightSignificantBytes := byteutil.ExtractSignificantBytes(rightDirect)

		// Extract significant bytes from left (bytes from first non-zero to end)
		leftSignificantBytes := byteutil.ExtractSignificantBytes(leftDirect)

		// push: add bytes from right at the beginning, remove bytes from end of left
		// Concatenate: right bytes + left bytes
		result := append(rightSignificantBytes, leftSignificantBytes...)

		// If result exceeds 8 bytes, keep only the first 8 bytes
		if len(result) > 8 {
			result = result[:8]
		}

		// Pad to exactly 8 bytes with right-alignment
		result = byteutil.Padding64Bits(result)

		e.envpool.SetTemp(l, result)
		e.cursor++
		return nil
	}

	if op == emitter.OpHead {
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)

		// Ensure left is exactly 8 bytes
		leftDirect := byteutil.Padding64Bits(left)

		// Get index in bytes (not in 8-byte slots)
		index := int(byteutil.ToUint64(right))

		// Apply modulo 8 to handle any index value (handles negative values too)
		index = (index%8 + 8) % 8

		// Extract first 'index' bytes
		result := leftDirect[:index]

		// Pad to 8 bytes with right-alignment
		result = byteutil.Padding64Bits(result)

		e.envpool.SetTemp(l, result)
		e.cursor++
		return nil
	}

	if op == emitter.OpTail {
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)

		// Ensure left is exactly 8 bytes
		leftDirect := byteutil.Padding64Bits(left)

		// Get index in bytes (not in 8-byte slots)
		index := int(byteutil.ToUint64(right))

		// Apply modulo 8 to handle any index value (handles negative values too)
		index = (index%8 + 8) % 8

		// Extract bytes from index to end
		result := leftDirect[index:]

		// Pad to 8 bytes with right-alignment
		result = byteutil.Padding64Bits(result)

		e.envpool.SetTemp(l, result)
		e.cursor++
		return nil
	}

	if op == emitter.OpResult {
		l := byteutil.ToHex(label)
		if len(e.result) > 0 {
			tr := e.result
			tv := tr[len(tr)-1]
			Print(os.Stdout, e.debug, e.counter, op, l, tv, nil)
			e.envpool.SetTemp(l, tv)
			e.result = tr[:len(tr)-1]
		} else {
			Print(os.Stdout, e.debug, e.counter, op, l, nil, nil)
		}
		e.cursor++
		return nil
	}

	if op == emitter.OpGetArg {
		index := byteutil.ToUint64(left)
		v := e.envpool.QueryArgument(index)
		l := byteutil.ToHex(label)
		Print(os.Stdout, e.debug, e.counter, op, index, v, nil)
		e.envpool.SetTemp(l, v)
		e.cursor++
		return nil
	}

	if op == emitter.OpPushArg {
		Print(os.Stdout, e.debug, e.counter, op, left, nil, nil)
		e.envpool.Current().PushArgument(left)
		e.cursor++
		return nil
	}

	if op == emitter.OpIdent {
		k := byteutil.ToHex(left)
		Print(os.Stdout, e.debug, e.counter, op, string(left), right, nil)
		if v := e.envpool.GetLocal(k); v != nil {
			return fmt.Errorf("conflict between identifiers named %s", left)
		}
		if len(right) > 0 {
			e.envpool.SetLocal(k, right)
			e.cursor++
			return nil
		}
		return fmt.Errorf("identifier %s cannot be void", left)
	}

	if op == emitter.OpIf {
		e.envpool.Ahead()
		test := byteutil.ToBoolean(left)
		Print(os.Stdout, e.debug, e.counter, op, test, nil, nil)
		if test {
			e.cursor++
			return nil
		}
		e.cursor = e.cursor + byteutil.ToUint64(right) + 1
		return nil
	}

	if op == emitter.OpJump {
		Print(os.Stdout, e.debug, e.counter, op, left, nil, nil)
		e.cursor = e.cursor + binary.BigEndian.Uint64(left) + 1
		return nil
	}

	if op == emitter.OpLoad {
		k := byteutil.ToHex(left)
		Print(os.Stdout, e.debug, e.counter, op, string(left), nil, nil)
		l := byteutil.ToHex(label)
		if v := e.envpool.QueryLocal(k); v != nil {
			e.envpool.SetTemp(l, v)
			e.cursor++
			return nil
		}
		return fmt.Errorf("identifier %s not defined", left)
	}

	if op == emitter.OpBeginScope { // Open scope for block
		key := byteutil.ToHex(left)
		Print(os.Stdout, e.debug, e.counter, op, label, key, nil)
		start := uint64(e.cursor) + 1
		end := byteutil.ToUint64(right)
		if curr := e.envpool.Current(); curr != nil {
			insts := e.insts[start : start+end]
			curr.SetScopeCallable(key, insts, start, end)
		}
		e.cursor = e.cursor + end + 1
		return nil
	}

	if op == emitter.OpPrint {
		Print(os.Stdout, e.debug, e.counter, op, left, nil, nil)
		builtin.PrintFunction(left)
		e.cursor++
		return nil
	}

	if op == emitter.OpEcho {
		Print(os.Stdout, e.debug, e.counter, op, left, nil, nil)
		builtin.EchoFunction(left)
		e.cursor++
		return nil
	}

	if op == emitter.OpAssert {
		// OpAssert receives the label of the comparison result
		// We need to get the result from the temp storage
		// left is the label (e.g., "3032"), we need to convert it to hex and get from temp
		// right contains the line number (stored as uint64 bytes)
		l := byteutil.ToHex(left)
		r := e.envpool.GetTemp(l)
		line := byteutil.ToUint64(byteutil.Padding64Bits(right))

		if r == nil {
			// Collect error instead of returning immediately
			e.assertErrors = append(e.assertErrors, fmt.Sprintf("assertion failed: could not find comparison result on line %d", line))
			e.cursor++
			return nil
		}

		isTrue := byteutil.ToBoolean(r)
		Print(os.Stdout, e.debug, e.counter, op, r, nil, nil)
		if !isTrue {
			// Collect error instead of returning immediately
			e.assertErrors = append(e.assertErrors, fmt.Sprintf("assertion failed: expected condition to be true on line %d", line))
		}
		e.cursor++
		return nil
	}

	if op == emitter.OpPreCall {
		k := byteutil.ToHex(left)
		v := e.envpool.QueryLocal(k)
		Print(os.Stdout, e.debug, e.counter, op, string(left), v, nil)

		if v == nil {
			return fmt.Errorf("identifier %s not defined", left)
		}

		k = byteutil.ToHex(v)
		currseg := e.envpool.QueryScopeCallable(k)
		if currseg == nil {
			return fmt.Errorf("identifier %s is not callable segment", left)
		}
		e.currseg = currseg

		e.envpool.Ahead()

		e.cursor++

		return nil
	}

	if op == emitter.OpCall {
		Print(os.Stdout, e.debug, e.counter, op, string(left), nil, nil)
		e.envpool.SetContext(e.cursor+1, e.insts)
		e.cursor = 0
		e.insts = e.currseg.GetInstructions() // Retrieve instructions from function segment
		return nil
	}

	if op == emitter.OpReturn {
		Print(os.Stdout, e.debug, e.counter, op, left, nil, nil)
		if len(left) > 0 {
			e.result = append(e.result, left)
		}
		ctx := e.envpool.GetContext()
		e.envpool.Back()
		if ctx != nil {
			e.currseg = nil
			e.insts = ctx.GetInstructions()
			e.cursor = ctx.GetCursor()
		} else {
			e.cursor++
		}
		return nil
	}

	if op == emitter.OpOr {
		a := byteutil.ToBoolean(left)
		b := byteutil.ToBoolean(right)
		test := a || b
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		l := byteutil.ToHex(label)
		if test {
			e.envpool.SetTemp(l, byteutil.True)
		} else {
			e.envpool.SetTemp(l, byteutil.False)
		}
		e.cursor++
		return nil
	}

	if op == emitter.OpAnd {
		a := byteutil.ToBoolean(left)
		b := byteutil.ToBoolean(right)
		test := a && b
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		l := byteutil.ToHex(label)
		if test {
			e.envpool.SetTemp(l, byteutil.True)
		} else {
			e.envpool.SetTemp(l, byteutil.False)
		}
		e.cursor++
		return nil
	}

	// Convert byte arrays to uint64 for arithmetic/comparison operations
	// Note: Only first 8 bytes are used; larger arrays are truncated
	a := bytesToUint64ForArithmetic(left)
	b := bytesToUint64ForArithmetic(right)

	if op == emitter.OpEquals {
		r := byteutil.False
		test := a == b
		if test {
			r = byteutil.True
		}
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpDiff {
		r := byteutil.False
		test := a != b
		if test {
			r = byteutil.True
		}
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpBigger {
		r := byteutil.False
		test := a > b
		if test {
			r = byteutil.True
		}
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpSmaller {
		r := byteutil.False
		test := a < b
		if test {
			r = byteutil.True
		}
		Print(os.Stdout, e.debug, e.counter, op, test, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpMultiply {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		Print(os.Stdout, e.debug, e.counter, op, r, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		Print(os.Stdout, e.debug, e.counter, op, r, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpSubtract {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		Print(os.Stdout, e.debug, e.counter, op, r, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpDivide {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		Print(os.Stdout, e.debug, e.counter, op, r, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	if op == emitter.OpExponential {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		Print(os.Stdout, e.debug, e.counter, op, r, a, b)
		e.result = append(e.result, r)
		e.cursor++
		return nil
	}

	e.cursor++
	return nil
}

func (e *Evaluator) Evaluate(insts []emitter.Instruction) (map[string][]byte, error) {
	var err error
	e.insts = insts
	e.assertErrors = make([]string, 0) // Reset assert errors buffer

	for int(e.cursor) < len(e.insts) {
		if e.player != nil {
			for e.player.scanner.Scan() {
				fmt.Printf("[Cursor]: %v\n", e.cursor)
				break
			}
		}
		inst := e.insts[e.cursor]
		err = e.exec(inst.GetLabel(), inst.GetOpCode(), inst.GetLeft(), inst.GetRight())
		if err != nil {
			break
		}
	}

	e.cursor = 0
	temps := e.envpool.Current().Temps()
	e.envpool.Current().ClearTemps()

	return temps, err
}

func (e *Evaluator) GetAssertErrors() []string {
	return e.assertErrors
}

func NewWithPlayer(debug bool, player *Player) *Evaluator {
	return &Evaluator{
		player:       player,
		envpool:      environ.NewPool(environ.New(nil)),
		cursor:       0,
		insts:        make([]emitter.Instruction, 0),
		result:       make([][]byte, 0),
		counter:      new(uint64),
		debug:        debug,
		assertErrors: make([]string, 0),
	}
}

func New(debug bool) *Evaluator {
	return NewWithPlayer(debug, nil)
}
