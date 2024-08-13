package evaluator

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/guiferpa/aurora/emitter"
)

type Evaluator struct {
	opcodes []emitter.OpCode
	mem     map[string][]byte
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

func (e *Evaluator) exec(l, op, left, right []byte) {
	a := binary.BigEndian.Uint64(left)
	b := binary.BigEndian.Uint64(right)
	veb := op[7] // Verificator byte
	if veb == emitter.OpMul {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a*b)
		e.mem[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpAdd {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a+b)
		e.mem[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpSub {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a-b)
		e.mem[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpDiv {
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, a/b)
		e.mem[fmt.Sprintf("%x", l)] = r
	}
	if veb == emitter.OpExp {
		r := make([]byte, 8)
		v := math.Pow(float64(a), float64(b))
		binary.BigEndian.PutUint64(r, uint64(v))
		e.mem[fmt.Sprintf("%x", l)] = r
	}
}

func (e *Evaluator) Evaluate() {
	for _, oc := range e.opcodes {
		if oc.Operation[7] == 0x0 {
			e.mem[fmt.Sprintf("%x", oc.Label)] = oc.Left
			continue
		}
		left := oc.Left
		if e.IsReference(left) {
			pleft := left
			left = e.mem[fmt.Sprintf("%x", pleft)]
			delete(e.mem, fmt.Sprintf("%x", pleft))
		}
		right := oc.Right
		if e.IsReference(right) {
			pright := right
			right = e.mem[fmt.Sprintf("%x", pright)]
			delete(e.mem, fmt.Sprintf("%x", pright))
		}
		e.exec(oc.Label, oc.Operation, left, right)
	}
}

func (e *Evaluator) GetMemory() map[string][]byte {
	return e.mem
}

func New(ocs []emitter.OpCode) *Evaluator {
	return &Evaluator{ocs, make(map[string][]byte)}
}
