package evaluator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
)

type Evaluator struct {
	envpool *environ.Pool
	params  [][]byte
	insts   []emitter.Instruction
	cursor  int
	result  []byte
	temps   map[string][]byte
	labels  map[string]int
}

func isTemp(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[len(bs)-1] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) walkTemps(bs []byte) []byte {
	pbs := bs
	bs = e.temps[fmt.Sprintf("%x", pbs)]
	delete(e.temps, fmt.Sprintf("%x", pbs))
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
		if len(left) > 0 {
			e.temps[fmt.Sprintf("%x", label)] = left
		}
		e.cursor++
		return nil
	}
	if op == emitter.OpResult {
		if len(e.result) > 0 {
			e.temps[fmt.Sprintf("%x", label)] = e.result
		}
		e.cursor++
		return nil
	}
	if op == emitter.OpIdentify {
		k := fmt.Sprintf("%x", left)
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
	if op == emitter.OpBeginFunc {
		start := uint64(e.cursor) + 1
		end := byteutil.ToUint64(right)
		if curr := e.envpool.Current(); curr != nil {
			key := fmt.Sprintf("%x", left)
			insts := e.insts[start : end+1]
			curr.SetSegment(key, insts, start, end)
		}
		e.cursor = e.cursor + int(end) + 1
		return nil
	}
	if op == emitter.OpLoad {
		k := fmt.Sprintf("%x", left)
		if v := e.envpool.Query(k); v != nil {
			e.temps[fmt.Sprintf("%x", label)] = v
			e.cursor++
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s not defined", left))
	}

	if op == emitter.OpOBlock { // Open scope for block
		e.envpool.Ahead()
		e.cursor++
		return nil
	}

	if op == emitter.OpCBlock { // Close scope for block
		e.envpool.Back()
		e.cursor++
		return nil
	}

	if op == emitter.OpParameter {
		e.params = append(e.params, left)
		e.cursor++
		return nil
	}

	if op == emitter.OpPrint {
		builtin.PrintFunction(left)
		e.cursor++
		return nil
	}

	if op == emitter.OpReturn {
		if len(left) > 0 {
			e.result = left
		}
		e.envpool.Back()
		ctx := e.envpool.GetContext()
		e.insts = ctx.GetInstructions()
		e.cursor = ctx.GetCursor()
		return nil
	}

	if op == emitter.OpCall {
		params := e.params
		e.params = make([][]byte, 0)

		k := fmt.Sprintf("%x", left)
		v := e.envpool.Query(k)
		if v == nil {
			return errors.New(fmt.Sprintf("identifier %s not defined", left))
		}

		k = fmt.Sprintf("%x", v)
		segcurr := e.envpool.Current().GetSegment(k)
		if segcurr == nil {
			return errors.New(fmt.Sprintf("identifier %s is not callable segment", left))
		}

		e.envpool.SetContext(e.cursor+1, e.insts)

		e.cursor = 0
		e.insts = segcurr.GetInstructions() // Retrieve instructions from function segment

		e.envpool.Ahead()

		for _, p := range params {
			fmt.Println(p)
		}

		return nil
	}

	a := binary.BigEndian.Uint64(byteutil.Padding64Bits(left))
	b := binary.BigEndian.Uint64(byteutil.Padding64Bits(right))

	if op == emitter.OpEquals {
		r := make([]byte, 1)
		if a == b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpDiff {
		r := make([]byte, 1)
		if a != b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpBigger {
		r := make([]byte, 1)
		if a > b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpSmaller {
		r := make([]byte, 1)
		if a < b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}

	if op == emitter.OpMultiply {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpSubstract {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpDivide {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}
	if op == emitter.OpExponential {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		e.temps[fmt.Sprintf("%x", label)] = r
		e.cursor++
		return nil
	}

	e.cursor++
	return nil
}

func (e *Evaluator) Evaluate(insts []emitter.Instruction) (map[string][]byte, error) {
	var err error
	e.insts = insts
	for e.cursor < len(e.insts) {
		inst := e.insts[e.cursor]
		// fmt.Println(e.cursor, fmt.Sprintf("%x: %x %x %x", inst.GetLabel(), inst.GetOpCode(), inst.GetLeft(), inst.GetRight()))
		err = e.exec(inst.GetLabel(), inst.GetOpCode(), inst.GetLeft(), inst.GetRight())
		if err != nil {
			break
		}
	}
	e.cursor = 0
	labels := e.temps
	e.temps = make(map[string][]byte)
	return labels, err
}

func (e *Evaluator) GetEnvironPool() *environ.Pool {
	return e.envpool
}

func (e *Evaluator) GetInstructions() []emitter.Instruction {
	return e.insts
}

func New() *Evaluator {
	envpool := environ.NewPool(environ.New(nil))
	params := make([][]byte, 0)
	insts := make([]emitter.Instruction, 0)
	cursor := 0
	result := make([]byte, 0)
	temps := make(map[string][]byte, 0)
	labels := make(map[string]int)

	return &Evaluator{envpool, params, insts, cursor, result, temps, labels}
}
