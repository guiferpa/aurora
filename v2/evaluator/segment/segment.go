package segment

import (
	"github.com/guiferpa/aurora/emitter"
)

type Segment struct {
	insts []emitter.Instruction
}

func (seg *Segment) Add(ins emitter.Instruction) {
	seg.insts = append(seg.insts, ins)
}

func (seg *Segment) List() []emitter.Instruction {
	return seg.insts
}

func New() *Segment {
	insts := make([]emitter.Instruction, 0)
	return &Segment{insts}
}
