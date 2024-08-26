package environ

import "github.com/guiferpa/aurora/emitter"

type FunctionSegment struct {
	insts []emitter.Instruction
	begin uint64
	end   uint64
}

func (s *FunctionSegment) GetBegin() uint64 {
	return s.begin
}

func (s *FunctionSegment) GetEnd() uint64 {
	return s.end
}

func (s *FunctionSegment) GetInstructions() []emitter.Instruction {
	return s.insts
}
