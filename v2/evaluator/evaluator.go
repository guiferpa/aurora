package evaluator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/builtin"
	"github.com/guiferpa/aurora/evaluator/environ"
)

type Evaluator struct {
	envpool *environ.Pool
	params  [][]byte
	insts   []emitter.Instruction
	cursor  int
	temps   map[string][]byte
	labels  map[string]int
}

func Padding64Bits(bfs []byte) []byte {
	const size = 8
	if len(bfs) >= size {
		return bfs
	}
	bs := make([]byte, size)
	for i := 0; i < len(bfs); i++ {
		bs[(size-len(bfs))+i] = bfs[i]
	}
	return bs
}

func IsTemp(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[len(bs)-1] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) WalkTemps(bs []byte) []byte {
	pbs := bs
	bs = e.temps[fmt.Sprintf("%x", pbs)]
	delete(e.temps, fmt.Sprintf("%x", pbs))
	if IsTemp(bs) {
		return e.WalkTemps(bs)
	}
	return bs
}

func (e *Evaluator) exec(label []byte, op byte, left, right []byte) error {
	if IsTemp(left) {
		left = e.WalkTemps(left)
	}
	if IsTemp(right) {
		right = e.WalkTemps(right)
	}

	if op == emitter.OpSave {
		if len(left) > 0 {
			e.temps[fmt.Sprintf("%x", label)] = left
		}
		return nil
	}
	if op == emitter.OpIdentify {
		k := fmt.Sprintf("%x", left)
		if v := e.envpool.Current().Get(k); v != nil {
			return errors.New(fmt.Sprintf("conflict between identifiers named %s", left))
		}
		if len(right) > 0 {
			e.envpool.Set(k, right)
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s cannot be null", left))
	}
	if op == emitter.OpFunction {
		if len(right) > 0 {
			k := fmt.Sprintf("%x", left)
			e.envpool.Set(k, right)
		}
		return nil
	}
	if op == emitter.OpLoad {
		k := fmt.Sprintf("%x", left)
		if v := e.envpool.Query(k); v != nil {
			e.temps[fmt.Sprintf("%x", label)] = v
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s not defined", left))
	}

	if op == emitter.OpOBlock { // Open scope for block
		e.envpool.Ahead()
		return nil
	}

	if op == emitter.OpCBlock { // Close scope for block
		e.envpool.Back()
		return nil
	}

	if op == emitter.OpParameter {
		e.params = append(e.params, left)
		return nil
	}

	if op == emitter.OpPrint {
		builtin.PrintFunction(left)
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
		v = e.envpool.Query(k)
		if v == nil {
			return errors.New(fmt.Sprintf("identifier %s is not a function", left))
		}

		e.envpool.Ahead()

		for _, p := range params {
			fmt.Println(p)
		}

		e.envpool.Back()

		return nil
	}

	a := binary.BigEndian.Uint64(Padding64Bits(left))
	b := binary.BigEndian.Uint64(Padding64Bits(right))

	if op == emitter.OpEquals {
		r := make([]byte, 1)
		if a == b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpDiff {
		r := make([]byte, 1)
		if a != b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpBigger {
		r := make([]byte, 1)
		if a > b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpSmaller {
		r := make([]byte, 1)
		if a < b {
			r = []byte{1}
		}
		e.temps[fmt.Sprintf("%x", label)] = r
	}

	if op == emitter.OpMultiply {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpSubstract {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpDivide {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpExponential {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		e.temps[fmt.Sprintf("%x", label)] = r
	}
	return nil
}

func (e *Evaluator) Evaluate(insts []emitter.Instruction) (map[string][]byte, error) {
	e.insts = insts
	e.cursor = 0
	for e.cursor < len(insts) {
		ins := insts[e.cursor]
		if err := e.exec(ins.GetLabel(), ins.GetOpCode(), ins.GetLeft(), ins.GetRight()); err != nil {
			return nil, err
		}
		e.cursor++
	}
	labels := e.temps
	e.temps = make(map[string][]byte)
	return labels, nil
}

func (e *Evaluator) GetEnvironPool() *environ.Pool {
	return e.envpool
}

func (e *Evaluator) GetInstructions() []emitter.Instruction {
	return e.insts
}

func New() *Evaluator {
	pool := environ.NewPool(environ.New(nil))
	params := make([][]byte, 0)
	insts := make([]emitter.Instruction, 0)
	cursor := 0
	temps := make(map[string][]byte, 0)
	labels := make(map[string]int)

	return &Evaluator{pool, params, insts, cursor, temps, labels}
}
