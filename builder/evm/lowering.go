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

// IsBinaryValueConsumer returns true for ops that consume two value operands (left/right) from the operand map.
// Used so we only reorder for Add/Sub/Mul/Div; OpReturn, OpBeginScope, etc. are left as-is and not merged.
func IsBinaryValueConsumer(op byte) bool {
	switch op {
	case emitter.OpAdd, emitter.OpSubtract, emitter.OpMultiply, emitter.OpDivide:
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

		if IsBinaryValueConsumer(inst.GetOpCode()) && okl && okr {
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
			// OpReturn consumes one value (return value); emit it before Return only when not already in out.
			// (After a binary op we set out = curb, so the result is already last in out; don't duplicate.)
			if inst.GetOpCode() == emitter.OpReturn && okr {
				lastIsBinary := len(out) > 0 && IsBinaryValueConsumer(out[len(out)-1].GetOpCode())
				if !lastIsBinary {
					out = append(out, or...)
				}
				delete(operands, lr)
			}
			out = append(out, inst)
		}
	}

	return out
}
