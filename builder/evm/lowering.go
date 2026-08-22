package evm

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// Putting the instructions of a scope in the order the stack needs.
//
// The IR names values: an instruction carries a label, and whoever uses that value names the
// label instead of holding it. The EVM has no names — it has a stack, and an operand has to be
// on top of it at the moment the instruction that eats it runs. This is where one becomes the
// other, and it is a phase rather than a trick: on a machine with registers the same crossing
// is called register allocation.
//
// It used to guess. Which operands named values was decided from the opcode, by a list of the
// four arithmetic instructions, and everything outside that list was emitted as it came — so
// an instruction whose value nobody had put on the stack got written anyway. The IR says what
// each operand is now, so this reads instead of remembering, and an opcode nobody thought of
// here is handled by the same rule as the rest.

// consumes answers the operands that name values, in the order those values have to reach the
// stack.
//
// A Ref names a value another instruction left behind; nothing else does. An Imm is the value
// itself and a Const belongs to the operation, and neither is waiting on the stack.
//
// Subtraction and division read theirs the other way round: the EVM computes `top - next`, so
// the right one is pushed first. That is the machine's, not the IR's, which is why it is the
// only thing here still keyed by opcode.
func consumes(inst ir.Instruction) []ir.Operand {
	taken := make([]ir.Operand, 0, 2)
	for _, operand := range inst.GetOperands() {
		if operand.Kind() == ir.KindRef {
			taken = append(taken, operand)
		}
	}
	if flipped(inst.GetOpCode()) && len(taken) == 2 {
		taken[0], taken[1] = taken[1], taken[0]
	}
	return taken
}

// flipped names the operations the EVM reads the other way round.
func flipped(op byte) bool {
	return op == ir.OpSubtract || op == ir.OpDivide
}

// produces answers whether an instruction leaves its value on the stack, under its own label.
//
// A binding does not: its value goes to memory and the stack comes out as it went in. That is
// why it is not here, and why the one place a binding is read as a value — a scope whose last
// expression is one — is handled where that is known.
func produces(op byte) bool {
	switch op {
	case ir.OpSave, ir.OpGetFeed, ir.OpLoad, ir.OpAdd, ir.OpSubtract, ir.OpMultiply, ir.OpDivide:
		return true
	default:
		return false
	}
}

// Lowering answers the instructions of one scope in the order the stack needs them.
func Lowering(insts []ir.Instruction, tapeSize int) []ir.Instruction {
	if len(insts) < 2 {
		return insts
	}
	return ResolveOperandsOrder(insts, tapeSize)
}

// ResolveOperandsOrder emits every value right before whoever takes it.
//
// An instruction that leaves a value is held back under its label rather than emitted where it
// was written: the code that produces it belongs next to the code that eats it. Holding it
// back is safe because everything held back is pure — a constant, an argument, a read of
// memory, or arithmetic over those — so moving it moves nothing observable.
//
// A value nobody takes has nowhere to be moved to, so it stays where it was written. That is
// the last expression of a scope, which is the scope's answer.
func ResolveOperandsOrder(insts []ir.Instruction, tapeSize int) []ir.Instruction {
	taken := labelsTaken(insts)
	pending := make(map[string][]ir.Instruction)
	out := make([]ir.Instruction, 0, len(insts))

	for _, inst := range insts {
		op := inst.GetOpCode()

		sequence := make([]ir.Instruction, 0, 3)
		for _, operand := range consumes(inst) {
			label := byteutil.ToHex(operand.Bytes())
			// A label with nothing under it is a value from outside this scope, or one
			// already taken. Either way there is nothing to emit for it here.
			if produced, held := pending[label]; held {
				sequence = append(sequence, produced...)
				delete(pending, label)
			}
		}
		sequence = append(sequence, inst)

		label := byteutil.ToHex(inst.GetLabel())
		switch {
		case produces(op) && taken[label] > 0:
			pending[label] = sequence
		case produces(op):
			// Nobody takes it, so there is nowhere to move it to: it is the answer of the
			// scope it was written in, and it stays where it was written.
			out = append(out, sequence...)
		case op == ir.OpIdent && taken[label] > 0:
			// A scope whose last expression is a binding answers with the neutral value,
			// which is what the evaluator answers. On the stack that has to be pushed: the
			// binding itself left nothing there.
			sequence = append(sequence, ir.NewInstruction(inst.GetLabel(), ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))
			pending[label] = sequence
		default:
			out = append(out, sequence...)
		}
	}

	return out
}

// labelsTaken counts how many instructions ask for each value.
func labelsTaken(insts []ir.Instruction) map[string]int {
	taken := make(map[string]int)
	for _, inst := range insts {
		for _, operand := range consumes(inst) {
			taken[byteutil.ToHex(operand.Bytes())]++
		}
	}
	return taken
}

// StackDepth answers how deep the stack is after each instruction of a lowered scope, starting
// at zero.
//
// It is what says a scope is balanced, and it is what a jump will need: the two sides of a
// branch have to meet with the same stack under them.
func StackDepth(insts []ir.Instruction) []int {
	depth := make([]int, len(insts)+1)
	for at, inst := range insts {
		op := inst.GetOpCode()
		after := depth[at] - len(consumes(inst))
		if produces(op) {
			after++
		}
		depth[at+1] = after
	}
	return depth
}
