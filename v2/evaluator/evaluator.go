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
	envpool   *environ.Pool
	params    [][]byte
	functions map[string][]emitter.Instruction
	insts     []emitter.Instruction
	labels    map[string][]byte
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

func IsLabels(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[len(bs)-1] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) WalkLabels(bs []byte) []byte {
	pbs := bs
	bs = e.labels[fmt.Sprintf("%x", pbs)]
	delete(e.labels, fmt.Sprintf("%x", pbs))
	if IsLabels(bs) {
		return e.WalkLabels(bs)
	}
	return bs
}

func (e *Evaluator) exec(label []byte, op byte, left, right []byte) error {
	if IsLabels(left) {
		left = e.WalkLabels(left)
	}
	if IsLabels(right) {
		right = e.WalkLabels(right)
	}

	if op == emitter.OpLabel {
		e.labels[fmt.Sprintf("%x", label)] = left
		return nil
	}
	if op == emitter.OpIdentify { // Create a definition
		if len(right) > 0 {
			k := fmt.Sprintf("%x", left)
			e.envpool.Set(k, right)
		}
		return nil
	}
	if op == emitter.OpFunction {
		if len(right) > 0 {
			k := fmt.Sprintf("%x", left)
			e.envpool.Set(k, right)
		}
		return nil
	}
	if op == emitter.OpLoad { // Get a definition
		k := fmt.Sprintf("%x", left)
		if v := e.envpool.Query(k); v != nil {
			e.labels[fmt.Sprintf("%x", label)] = v
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
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpDiff {
		r := make([]byte, 1)
		if a != b {
			r = []byte{1}
		}
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpBigger {
		r := make([]byte, 1)
		if a > b {
			r = []byte{1}
		}
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpSmaller {
		r := make([]byte, 1)
		if a < b {
			r = []byte{1}
		}
		e.labels[fmt.Sprintf("%x", label)] = r
	}

	if op == emitter.OpMultiply {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpSubstract {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpDivide {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	if op == emitter.OpExponential {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		e.labels[fmt.Sprintf("%x", label)] = r
	}
	return nil
}

func (e *Evaluator) Evaluate(insts []emitter.Instruction) (map[string][]byte, error) {
	e.insts = insts
	for _, ins := range e.insts {
		if err := e.exec(ins.GetLabel(), ins.GetOpCode(), ins.GetLeft(), ins.GetRight()); err != nil {
			return nil, err
		}
	}
	labels := e.labels
	e.labels = make(map[string][]byte)
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
	functions := make(map[string][]emitter.Instruction, 0)
	opcodes := make([]emitter.Instruction, 0)
	labels := make(map[string][]byte, 0)

	return &Evaluator{pool, params, functions, opcodes, labels}
}
