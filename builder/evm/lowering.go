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
// other.
//
// It used to be a pattern match: two operands and an operator in the middle, with a special
// case for the one other instruction that takes a value. Everything else was emitted as it
// came, which is why `ident x = feed(0);` inside a scope compiled to an MSTORE with nothing
// under it — the value it was meant to store was never pushed, and the contract answered
// nothing at all, quietly.
//
// So the shape is written down instead of guessed. Each instruction says what it takes and
// whether it leaves anything, and the order falls out of that: an operand is emitted right
// before whoever asked for it. A jump needs exactly this to be possible — it has to know what
// is on the stack where the code splits.

// A field is where an instruction writes the label of a value it takes.
type field int

const (
	fieldLeft field = iota
	fieldRight
)

// consumes answers the fields of an instruction that name values it takes, in the order those
// values have to reach the stack.
//
// Subtraction and division read theirs the other way round: the EVM pops the left operand
// first and computes `top - next`, so the right one is pushed first.
func consumes(op byte) []field {
	switch op {
	case ir.OpAdd, ir.OpMultiply:
		return []field{fieldLeft, fieldRight}
	case ir.OpSubtract, ir.OpDivide:
		return []field{fieldRight, fieldLeft}
	case ir.OpIdent, ir.OpReturn:
		return []field{fieldRight}
	default:
		return nil
	}
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
// A value nobody takes has nowhere to be moved to, so it stays where it was written — that is
// the last expression of a scope, which is the scope's answer.
func ResolveOperandsOrder(insts []ir.Instruction, tapeSize int) []ir.Instruction {
	taken := labelsTaken(insts)
	pending := make(map[string][]ir.Instruction)
	out := make([]ir.Instruction, 0, len(insts))

	for _, inst := range insts {
		op := inst.GetOpCode()

		sequence := make([]ir.Instruction, 0, 3)
		for _, where := range consumes(op) {
			label := byteutil.ToHex(fieldOf(inst, where))
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
			sequence = append(sequence, ir.NewInstruction(inst.GetLabel(), ir.OpSave, byteutil.FalseTape(tapeSize), nil))
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
		for _, where := range consumes(inst.GetOpCode()) {
			taken[byteutil.ToHex(fieldOf(inst, where))]++
		}
	}
	return taken
}

// fieldOf reads the label an instruction wrote in one of its fields.
func fieldOf(inst ir.Instruction, where field) []byte {
	if where == fieldLeft {
		return inst.GetLeft()
	}
	return inst.GetRight()
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
		after := depth[at] - len(consumes(op))
		if produces(op) {
			after++
		}
		depth[at+1] = after
	}
	return depth
}
