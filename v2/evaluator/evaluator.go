package evaluator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/guiferpa/aurora/emitter"
)

type Evaluator struct {
	mem    map[string][]byte
	labels map[string][]byte
}

func (e *Evaluator) IsReference(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[6] == 0x74 { // t
		return true
	}
	return false
}

func (e *Evaluator) exec(l, op, left, right []byte) error {
	veb := op[7]              // Verificator byte
	if veb == emitter.OpPin { // Create a definition
		e.mem[fmt.Sprintf("%x", left)] = right
	}
	if veb == emitter.OpGet { // Get a definition
		if v, ok := e.mem[fmt.Sprintf("%x", left)]; ok {
			e.labels[fmt.Sprintf("%x", l)] = v
		} else {
			return errors.New(fmt.Sprintf("identifier %s not defined", left))
		}
	}

	a := binary.BigEndian.Uint64(left)
	b := binary.BigEndian.Uint64(right)
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
	for _, oc := range opcodes {
		if oc.Operation[7] == 0x0 {
			e.labels[fmt.Sprintf("%x", oc.Label)] = oc.Left
			continue
		}
		left := oc.Left
		if e.IsReference(left) {
			pleft := left
			left = e.labels[fmt.Sprintf("%x", pleft)]
			delete(e.labels, fmt.Sprintf("%x", pleft))
		}
		right := oc.Right
		if e.IsReference(right) {
			pright := right
			right = e.labels[fmt.Sprintf("%x", pright)]
			delete(e.labels, fmt.Sprintf("%x", pright))
		}
		if err := e.exec(oc.Label, oc.Operation, left, right); err != nil {
			return nil, err
		}
	}
	labels := e.labels
	e.labels = make(map[string][]byte)
	return labels, nil
}

func New() *Evaluator {
	return &Evaluator{make(map[string][]byte), make(map[string][]byte)}
}
