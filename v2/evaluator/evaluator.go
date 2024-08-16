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
	functions map[string][]emitter.OpCode
	opcodes   []emitter.OpCode
	labels    map[string][]byte
}

func IsLabel(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[len(bs)-1] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) WalkLabel(bs []byte) []byte {
	pbs := bs
	bs = e.labels[fmt.Sprintf("%x", pbs)]
	delete(e.labels, fmt.Sprintf("%x", pbs))
	if IsLabel(bs) {
		return e.WalkLabel(bs)
	}
	return bs
}

func (e *Evaluator) exec(l, op, left, right []byte) error {
	if IsLabel(left) {
		left = e.WalkLabel(left)
	}
	if IsLabel(right) {
		right = e.WalkLabel(right)
	}

	veb := op[7] // Verificator byte

	if veb == emitter.OpLab {
		e.labels[fmt.Sprintf("%x", l)] = left
		return nil
	}
	if veb == emitter.OpPin { // Create a definition
		if len(right) > 0 {
			k := fmt.Sprintf("%x", left)
			e.envpool.Set(k, right)
		}
		return nil
	}
	if veb == emitter.OpFun {
		if len(right) > 0 {
			k := fmt.Sprintf("%x", left)
			e.envpool.Set(k, right)
		}
		return nil
	}
	if veb == emitter.OpGet { // Get a definition
		k := fmt.Sprintf("%x", left)
		if v := e.envpool.Query(k); v != nil {
			e.labels[fmt.Sprintf("%x", l)] = v
			return nil
		}
		return errors.New(fmt.Sprintf("identifier %s not defined", left))
	}

	if veb == emitter.OpOBl { // Open scope for block
		e.envpool.Ahead()
		return nil
	}

	if veb == emitter.OpCBl { // Close scope for block
		e.envpool.Back()
		return nil
	}

	if veb == emitter.OpPar {
		fmt.Println(fmt.Sprintf("%x", left), fmt.Sprintf("%s", left))
		e.params = append(e.params, left)
		return nil
	}

	if veb == emitter.OpPrt {
		builtin.PrintFunction(left)
		return nil
	}

	if veb == emitter.OpCal {
		e.envpool.Ahead()
		fmt.Println(e.params)
		e.envpool.Back()
		e.params = make([][]byte, 0)
		return nil
	}

	a := binary.BigEndian.Uint64(left)
	b := binary.BigEndian.Uint64(right)

	if veb == emitter.OpEqu {
		r := make([]byte, 8)
		if a == b {
			r = []byte{0, 0, 0, 0, 0, 0, 0, 1}
		}
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpDif {
		r := make([]byte, 8)
		if a != b {
			r = []byte{0, 0, 0, 0, 0, 0, 0, 1}
		}
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpBig {
		r := make([]byte, 8)
		if a > b {
			r = []byte{0, 0, 0, 0, 0, 0, 0, 1}
		}
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpSma {
		r := make([]byte, 8)
		if a < b {
			r = []byte{0, 0, 0, 0, 0, 0, 0, 1}
		}
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpMul {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpSub {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpDiv {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpExp {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		e.labels[fmt.Sprintf("%x", l)] = r
	}
	return nil
}

func (e *Evaluator) Evaluate(opcodes []emitter.OpCode) (map[string][]byte, error) {
	e.opcodes = opcodes
	for _, oc := range e.opcodes {
		if err := e.exec(oc.Label, oc.Operation, oc.Left, oc.Right); err != nil {
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

func (e *Evaluator) GetOpCodes() []emitter.OpCode {
	return e.opcodes
}

func New() *Evaluator {
	pool := environ.NewPool(environ.New(nil))
	params := make([][]byte, 0)
	functions := make(map[string][]emitter.OpCode, 0)
	opcodes := make([]emitter.OpCode, 0)
	labels := make(map[string][]byte, 0)

	return &Evaluator{pool, params, functions, opcodes, labels}
}
