package environ

import "github.com/guiferpa/aurora/emitter"

type ScopeCallable struct {
	insts []emitter.Instruction
	begin uint64
	end   uint64
}

func (s *ScopeCallable) GetBegin() uint64 {
	return s.begin
}

func (s *ScopeCallable) GetEnd() uint64 {
	return s.end
}

func (s *ScopeCallable) GetInstructions() []emitter.Instruction {
	return s.insts
}
