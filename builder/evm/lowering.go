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

// valueOperands answers the operands that are values, in the order they have to reach the
// stack.
//
// A Ref names a value another instruction left behind and an Imm is the value itself, written
// where it is used. Both end up on the stack; what differs is who puts them there — a Ref is
// put there by the instruction that produced it, and an Imm by the instruction that takes it.
//
// A Const is not here: it belongs to the operation rather than to the program, and the writer
// reads it out of the instruction without the stack being involved.
//
// Subtraction and division read theirs the other way round: the EVM computes `top - next`, so
// the right one reaches the stack first. That is the machine's, not the IR's, which is why it
// is the only thing here still keyed by opcode.
func valueOperands(inst ir.Instruction) []ir.Operand {
	values := make([]ir.Operand, 0, 2)
	for _, operand := range inst.GetOperands() {
		if kind := operand.Kind(); kind == ir.KindRef || kind == ir.KindImm {
			values = append(values, operand)
		}
	}
	if flipped(inst.GetOpCode()) && len(values) == 2 {
		values[0], values[1] = values[1], values[0]
	}
	return values
}

// consumes answers the values this instruction waits for on the stack: the ones another
// instruction has to leave there. An Imm is not waited for — it is written down.
func consumes(inst ir.Instruction) []ir.Operand {
	taken := make([]ir.Operand, 0, 2)
	for _, operand := range valueOperands(inst) {
		if operand.Kind() == ir.KindRef {
			taken = append(taken, operand)
		}
	}
	return taken
}

// flipped names the operations the EVM reads top-first.
//
// It pops its first operand off the top, so `a - b` wants b underneath a: the right one
// reaches the stack first. The same goes for division, for raising to a power, and for the two
// comparisons that are not symmetric — `a bigger b` is GT of a over b, and GT reads the top as
// the left-hand side.
//
// Equality, difference, and and or are symmetric, so the order they arrive in says nothing.
func flipped(op byte) bool {
	switch op {
	case ir.OpSubtract, ir.OpDivide, ir.OpExponential, ir.OpBigger, ir.OpSmaller:
		return true
	default:
		return false
	}
}

// produces answers whether an instruction leaves its value on the stack, under its own label.
//
// A binding does not: its value goes to memory and the stack comes out as it went in. That is
// why it is not here, and why the one place a binding is read as a value — a scope whose last
// expression is one — is handled where that is known.
func produces(op byte) bool {
	switch op {
	case ir.OpSave, ir.OpGetFeed, ir.OpLoad,
		ir.OpAdd, ir.OpSubtract, ir.OpMultiply, ir.OpDivide, ir.OpExponential,
		ir.OpEquals, ir.OpDiff, ir.OpBigger, ir.OpSmaller, ir.OpAnd, ir.OpOr,
		ir.OpJoin, ir.OpField:
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
// divides answers whether control can leave an instruction for somewhere other than the next
// one, or arrive at it from somewhere other than the one before.
//
// Nothing is moved across one. A value held back is emitted right before whoever takes it,
// which is sound while both sit on the same straight run of instructions — past a branch it is
// not: the value would be produced only when that side runs, and whoever takes it may be
// reached the other way. It is the rule every compiler keeps, and it is why a scheduler works
// inside a block and never through one.
func divides(op byte) bool {
	switch op {
	case ir.OpIf, ir.OpJump, ir.OpReturn, ir.OpBeginScope, ir.OpDefer, ir.OpCall:
		return true
	default:
		return false
	}
}

// ResolveOperandsOrder emits every value right before whoever takes it.
//
// An instruction that leaves a value is held back under its label rather than emitted where it
// was written: the code that produces it belongs next to the code that eats it. Holding it
// back is safe because everything held back is pure — a constant, an argument, a read of
// memory, or arithmetic over those — so moving it moves nothing observable.
//
// It is held back only as far as the next place control can go. Anything still waiting when
// one of those is reached is emitted first, in the order it was written, because past that
// point there is no telling whether it would have run.
//
// A value nobody takes has nowhere to be moved to, so it stays where it was written. That is
// the last expression of a scope, which is the scope's answer.
func ResolveOperandsOrder(insts []ir.Instruction, tapeSize int) []ir.Instruction {
	taken := labelsTaken(insts)
	pending := make(map[string][]ir.Instruction)
	// The order they were held back in, so flushing them keeps it.
	waiting := make([]string, 0, len(insts))
	out := make([]ir.Instruction, 0, len(insts))

	flush := func() {
		for _, label := range waiting {
			if held, ok := pending[label]; ok {
				out = append(out, held...)
				delete(pending, label)
			}
		}
		waiting = waiting[:0]
	}

	hold := func(label string, sequence []ir.Instruction) {
		pending[label] = sequence
		waiting = append(waiting, label)
	}

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

		if divides(op) {
			// What this takes was just put in front of it, so it goes out whole — and
			// everything else waiting goes out ahead of it, where it is still on the
			// straight run it was written on.
			held := sequence
			flush()
			out = append(out, held...)
			continue
		}

		label := byteutil.ToHex(inst.GetLabel())
		switch {
		case produces(op) && taken[label] > 0:
			hold(label, sequence)
		case produces(op):
			// Nobody takes it, so there is nowhere to move it to: it is the answer of the
			// scope it was written in, and it stays where it was written.
			out = append(out, sequence...)
		case op == ir.OpIdent && taken[label] > 0:
			// A scope whose last expression is a binding answers with the neutral value,
			// which is what the evaluator answers. On the stack that has to be pushed: the
			// binding itself left nothing there.
			sequence = append(sequence, ir.NewInstruction(inst.GetLabel(), ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))
			hold(label, sequence)
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
