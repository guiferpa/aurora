package evaluator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
)

type Evaluator struct {
	player  *Player
	envpool *environ.Pool
	cursor  uint64
	insts   []emitter.Instruction
	currseg *environ.ScopeCallable
	result  [][]byte
	counter *uint64
	debug   bool
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
	l := fmt.Sprintf("%x", bs)
	bs = e.envpool.GetTemp(l)
	if isTemp(bs) {
		return e.walkTemps(bs)
	}
	return bs
}

func (e *Evaluator) exec(label []byte, op byte, left, right []byte) error {
	if isTemp(left) {
		left = e.walkTemps(left)
	}

	if isTemp(right) {
		right = e.walkTemps(right)
	}

	if op == emitter.OpSave {
		l := fmt.Sprintf("%x", label)
		Print(os.Stdout, e.debug, e.counter, op, left, right, nil)
		e.envpool.SetTemp(l, left)
		e.cursor++
		return nil
	}

	if op == emitter.OpResult {
		l := fmt.Sprintf("%x", label)
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
		l := fmt.Sprintf("%x", label)
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
		k := fmt.Sprintf("%x", left)
		Print(os.Stdout, e.debug, e.counter, op, fmt.Sprintf("%s", left), right, nil)
		if v := e.envpool.GetLocal(k); v != nil {
			return errors.New(fmt.Sprintf("conflict between identifiers named %s", left))
		}
		if len(right) > 0 {
			e.envpool.SetLocal(k, right)
			e.cursor++
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s cannot be void", left))
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
		k := fmt.Sprintf("%x", left)
		Print(os.Stdout, e.debug, e.counter, op, fmt.Sprintf("%s", left), nil, nil)
		l := fmt.Sprintf("%x", label)
		if v := e.envpool.QueryLocal(k); v != nil {
			e.envpool.SetTemp(l, v)
			e.cursor++
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s not defined", left))
	}

	if op == emitter.OpBeginScope { // Open scope for block
		key := fmt.Sprintf("%x", left)
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

	if op == emitter.OpPreCall {
		k := fmt.Sprintf("%x", left)
		v := e.envpool.QueryLocal(k)
		Print(os.Stdout, e.debug, e.counter, op, fmt.Sprintf("%s", left), v, nil)

		if v == nil {
			return errors.New(fmt.Sprintf("identifier %s not defined", left))
		}

		k = fmt.Sprintf("%x", v)
		currseg := e.envpool.QueryScopeCallable(k)
		if currseg == nil {
			return errors.New(fmt.Sprintf("identifier %s is not callable segment", left))
		}
		e.currseg = currseg

		e.envpool.Ahead()

		e.cursor++

		return nil
	}

	if op == emitter.OpCall {
		Print(os.Stdout, e.debug, e.counter, op, fmt.Sprintf("%s", left), nil, nil)
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
		l := fmt.Sprintf("%x", label)
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
		l := fmt.Sprintf("%x", label)
		if test {
			e.envpool.SetTemp(l, byteutil.True)
		} else {
			e.envpool.SetTemp(l, byteutil.False)
		}
		e.cursor++
		return nil
	}

	a := binary.BigEndian.Uint64(byteutil.Padding64Bits(left))
	b := binary.BigEndian.Uint64(byteutil.Padding64Bits(right))

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

func (e *Evaluator) GetEnvironPool() *environ.Pool {
	return e.envpool
}

func (e *Evaluator) GetInstructions() []emitter.Instruction {
	return e.insts
}

func NewWithPlayer(debug bool, player *Player) *Evaluator {
	return &Evaluator{
		player:  player,
		envpool: environ.NewPool(environ.New(nil)),
		cursor:  0,
		insts:   make([]emitter.Instruction, 0),
		result:  make([][]byte, 0),
		counter: new(uint64),
		debug:   debug,
	}
}

func New(debug bool) *Evaluator {
	return NewWithPlayer(debug, nil)
}
