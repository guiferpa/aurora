package evaluator

import (
	"github.com/guiferpa/aurora/emitter"
)

type Evaluator struct {
	opcodes []emitter.OpCode
	temps   [][]byte
}

func (e *Evaluator) Evaluate()  {}

func New(ocs []emitter.OpCode) *Evaluator {
	return &Evaluator{ocs, make([][]byte, 0)}
}
