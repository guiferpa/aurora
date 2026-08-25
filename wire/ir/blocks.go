package ir

// Operations over blocks. They are here and not with whoever joins programs because they are
// arithmetic over the IR and nothing else — and getting any of them wrong is quiet.

// Shifted answers the same blocks, numbered from an offset.
//
// A file is compiled on its own and numbers its blocks from zero, so putting two files in one
// program means one of them has to move. Everything that names a block moves with it: where a
// terminator goes, and the value a binding holds when a name was bound to a scope.
//
// It is here rather than in whoever joins them because it is arithmetic over the IR and
// nothing else. Getting it wrong is quiet — a block that exists, named by mistake.
func Shifted(blocks []Block, by BlockID) []Block {
	moved := make([]Block, 0, len(blocks))
	for _, block := range blocks {
		block.ID += by
		block.Insts = shiftedInsts(block.Insts, by)
		block.Term = shiftedTerm(block.Term, by)
		moved = append(moved, block)
	}
	return moved
}

func shiftedInsts(insts []Instruction, by BlockID) []Instruction {
	moved := make([]Instruction, 0, len(insts))
	for _, inst := range insts {
		operands := inst.GetOperands()
		shifted := make([]Operand, 0, len(operands))
		for _, operand := range operands {
			if operand.Kind() == KindBlock {
				operand = BlockOf(operand.Block() + by)
			}
			shifted = append(shifted, operand)
		}
		moved = append(moved, NewInstructionOver(
			inst.GetLabel(), inst.GetOpCode(), shifted...).At(inst.GetOrigin()))
	}
	return moved
}

func shiftedTerm(term Terminator, by BlockID) Terminator {
	targets := make([]Target, 0, len(term.Targets))
	for _, target := range term.Targets {
		target.Block += by
		targets = append(targets, target)
	}
	term.Targets = targets
	return term
}

// Reaches answers the blocks control can get to from one, itself included.
//
// A terminator is the only thing that moves control, so this is the whole answer — and a scope
// written inside is not in it, because a binding names a block and naming is not going.
func Reaches(blocks []Block, from BlockID) []BlockID {
	seen := make(map[BlockID]bool, len(blocks))
	reached := make([]BlockID, 0, len(blocks))

	var walk func(id BlockID)
	walk = func(id BlockID) {
		if seen[id] || int(id) >= len(blocks) {
			return
		}
		seen[id] = true
		reached = append(reached, id)
		for _, target := range blocks[id].Term.Targets {
			walk(target.Block)
		}
	}
	walk(from)
	return reached
}

// GoesOnTo answers the same blocks, with everywhere the run from one of them ended now carrying
// on to another.
//
// It is how one file follows another. Each is compiled as a program of its own and ends by
// answering, and a program made of several runs them in the order their dependencies were
// found — so every ending in the first is turned into going to the second, and only the last
// file still answers.
//
// A scope is untouched by this: its blocks are not reached from the top of a file, because a
// binding names a block and naming is not going. So a scope still ends by answering, which is
// what a call needs it to do.
func GoesOnTo(blocks []Block, from, next BlockID) []Block {
	for _, id := range Reaches(blocks, from) {
		if blocks[id].Term.Kind == Ret {
			blocks[id].Term = Goes(next)
		}
	}
	return blocks
}
