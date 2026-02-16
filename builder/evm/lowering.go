package evm

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

func Lowering(insts []emitter.Instruction) []emitter.Instruction {
	if len(insts) == 0 {
		return insts
	}
	return ResolveOperandsOrder(insts)
}

func IsOperand(op byte) bool {
	switch op {
	case emitter.OpGetArg, emitter.OpSave, emitter.OpLoad:
		return true
	default:
		return false
	}
}

func IsAssociativeOperator(op byte) bool {
	switch op {
	case emitter.OpSubtract, emitter.OpDivide:
		return true
	default:
		return false
	}
}

func OperandStackDelta(op byte) int {
	if IsOperand(op) {
		return 1
	}
	if IsAssociativeOperator(op) {
		return -1
	}
	return 0
}

func GetOperandStackDeltaDepth(insts []emitter.Instruction) []int {
	depth := make([]int, len(insts)+1)
	depth[0] = 0
	for at, inst := range insts {
		depth[at+1] = depth[at] + OperandStackDelta(inst.GetOpCode())
	}
	return depth
}

func ResolveOperandsOrder(insts []emitter.Instruction) []emitter.Instruction {
	if len(insts) < 2 {
		return insts
	}
	operands := make(map[string][]emitter.Instruction, 0)
	out := make([]emitter.Instruction, 0)
	for _, inst := range insts {
		if IsOperand(inst.GetOpCode()) {
			label := byteutil.ToHex(inst.GetLabel())
			operands[label] = []emitter.Instruction{inst}
			continue
		}

		ll := byteutil.ToHex(inst.GetLeft())
		lr := byteutil.ToHex(inst.GetRight())
		ld := byteutil.ToHex(inst.GetLabel())

		ol, okl := operands[ll]
		or, okr := operands[lr]

		if okl && okr {
			curb := make([]emitter.Instruction, 0)

			if IsAssociativeOperator(inst.GetOpCode()) {
				curb = append(curb, or...)
				curb = append(curb, ol...)
			} else {
				curb = append(curb, ol...)
				curb = append(curb, or...)
			}
			curb = append(curb, inst)
			operands[ld] = curb

			delete(operands, ll)
			delete(operands, lr)

			out = curb
		} else {
			out = append(out, inst)
		}
	}

	return out
}
