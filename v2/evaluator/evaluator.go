package evaluator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/guiferpa/aurora/emitter"
)

type Evaluator struct {
	mem     map[string][]byte
	opcodes []emitter.OpCode
	labels  map[string][]byte
}

func (e *Evaluator) IsLabel(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[6] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) WalkLabel(bs []byte) []byte {
	pbs := bs
	bs = e.labels[fmt.Sprintf("%x", pbs)]
	delete(e.labels, fmt.Sprintf("%x", pbs))
	if e.IsLabel(bs) {
		return e.WalkLabel(bs)
	}
	return bs
}

func (e *Evaluator) exec(l, op, left, right []byte) error {
	veb := op[7] // Verificator byte

	if veb == emitter.OpPin { // Create a definition
		if len(right) > 0 {
			e.mem[fmt.Sprintf("%x", left)] = right
		}
		return nil
	}
	if veb == emitter.OpGet { // Get a definition
		if v, ok := e.mem[fmt.Sprintf("%x", left)]; ok {
			e.labels[fmt.Sprintf("%x", l)] = v
		} else {
			return errors.New(fmt.Sprintf("identifier %s not defined", left))
		}
	}

	if veb == emitter.OpOBl { // Open scope for block
		return nil
	}

	if veb == emitter.OpCBl { // Close scope for block
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
		if oc.Operation[7] == 0x0 {
			e.labels[fmt.Sprintf("%x", oc.Label)] = oc.Left
			continue
		}
		left := oc.Left
		if e.IsLabel(left) {
			left = e.WalkLabel(left)
		}
		right := oc.Right
		if e.IsLabel(right) {
			right = e.WalkLabel(right)
		}
		if err := e.exec(oc.Label, oc.Operation, left, right); err != nil {
			return nil, err
		}
	}
	labels := e.labels
	e.labels = make(map[string][]byte)
	return labels, nil
}

func (e *Evaluator) GetMemory() map[string][]byte {
	return e.mem
}

func (e *Evaluator) GetOpCodes() []emitter.OpCode {
	return e.opcodes
}

func New() *Evaluator {
	return &Evaluator{make(map[string][]byte), make([]emitter.OpCode, 0), make(map[string][]byte)}
}
