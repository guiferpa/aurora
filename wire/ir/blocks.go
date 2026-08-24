package ir

import "github.com/guiferpa/aurora/byteutil"

// BlocksOf answers the same program as blocks with terminators.
//
// The instruction list says structure by counting: an "if" carries how many instructions to
// skip, a deferred scope carries how long its body is, and where a scope ends is a return that
// names where it began. So the structure lives in the order of the list and in arithmetic over
// indices, and every consumer works it out again — the evaluator by moving a cursor, the
// builder by slicing and measuring. Two ways of counting the same thing is two chances to
// count it differently, and the builder has taken that chance more than once.
//
// This is the counting done once, where the structure was known in the first place. What comes
// out has no counts in it: a branch names the two blocks it chooses between, a scope is a
// block rather than a stretch of a list, and a block ends exactly one way.
//
// Five opcodes do not survive the crossing, because each of them was structure written as an
// instruction: OpDefer, OpBeginScope, OpIf, OpJump and OpReturn. What is left inside a block
// computes values and nothing else.
func BlocksOf(insts []Instruction) []Block {
	b := &blocking{scopes: make(map[string]BlockID)}
	top := b.reserve()
	// The top of a program takes values the way a scope does. Nothing applies any to it — no
	// transaction names it — so what it reads there is zeros, which is what reading past what
	// was applied answers. Saying how many it addresses is still what keeps the names it binds
	// from being kept in the same places.
	b.run(top, insts, feedsRead(insts), func() Terminator { return Ends(Nothing()) })
	return b.blocks
}

// blocking builds blocks as it walks, handing each the number it will be known by.
//
// A block is reserved before it is filled, because a branch names blocks that have not been
// walked yet: the arms of an "if" are named by the terminator that chooses between them, and
// that terminator is written before either arm is read.
type blocking struct {
	blocks []Block
	// scopes answers, for the label an OpDefer left, the block its body became — so the
	// binding that follows it can name a block instead of an instruction that is gone.
	scopes map[string]BlockID
}

// reserve takes the next number and leaves a place for the block to be put in.
func (b *blocking) reserve() BlockID {
	id := BlockID(len(b.blocks))
	b.blocks = append(b.blocks, Block{ID: id})
	return id
}

// put fills a block that was reserved.
func (b *blocking) put(id BlockID, params int, insts []Instruction, term Terminator, origin Origin) {
	b.blocks[id] = Block{ID: id, Params: params, Insts: insts, Term: term, Origin: origin}
}

// ends says how a run finishes when it reaches the end without a return of its own.
type ends func() Terminator

// run fills a reserved block from one straight run of instructions.
//
// It is one block until something splits it. A branch splits it into the run before, the two
// arms, and the block they meet at — and each of those is a straight run again, so the same
// walk handles a branch inside a branch without knowing that it is doing so.
func (b *blocking) run(id BlockID, insts []Instruction, params int, tail ends) {
	held := make([]Instruction, 0, len(insts))
	origin := Origin{}
	if len(insts) > 0 {
		origin = insts[0].GetOrigin()
	}

	for at := 0; at < len(insts); at++ {
		inst := insts[at]

		switch inst.GetOpCode() {
		case OpBeginScope:
			// Whatever opened this run was taken off before it got here, so anything that
			// opens now is a block written inside it.
			b.inline(id, params, held, origin, insts, at, tail)
			return

		case OpDefer:
			// The body is what this instruction says is its own, and it becomes a block.
			length := int(byteutil.ToUint64(inst.GetRight().Bytes()))
			body := insts[at+1 : at+1+length]
			scope := b.reserve()
			b.run(scope, opened(body), feedsRead(body), func() Terminator { return Ends(Nothing()) })
			b.scopes[byteutil.ToHex(inst.GetLabel())] = scope
			at += length

		case OpIdent:
			// A binding whose value was a scope names the block instead of an instruction
			// that is no longer there.
			if scope, ok := b.scopes[byteutil.ToHex(inst.GetRight().Bytes())]; ok {
				inst = NewInstruction(
					inst.GetLabel(), OpIdent, inst.GetLeft(), BlockOf(scope)).At(inst.GetOrigin())
			}
			held = append(held, inst)

		case OpReturn:
			// Whatever this closes was taken off before it got here, so a return that
			// arrives is one nothing claimed: the run ends, answering with what it names.
			b.put(id, params, held, Ends(inst.GetRight()), origin)
			return

		case OpIf:
			b.branch(id, params, held, origin, insts, at, tail)
			return

		default:
			held = append(held, inst)
		}
	}

	b.put(id, params, held, tail(), origin)
}

// inline splits a run at a block written inside it.
//
// A block is an expression: control goes into it, it computes a value, and control carries on
// with that value in hand. So it is three blocks — the run up to it, the block itself, and
// where the run carries on — and the value reaches the third the way the arms of a branch
// reach where they meet: handed over, rather than left in a place both sides agree on.
//
// It used to end the scope instead. A block opens with the instruction a scope's body opens
// with and ends with the same return, so the return of the inner one was read as the outer
// one's, and everything written after it was dropped from a contract that still deployed. What
// tells them apart is the name: a return names the thing it closes.
func (b *blocking) inline(id BlockID, params int, held []Instruction, origin Origin, insts []Instruction, at int, tail ends) {
	opened := byteutil.ToHex(insts[at].GetLabel())

	closes := len(insts)
	answer := Nothing()
	for ahead := at + 1; ahead < len(insts); ahead++ {
		if insts[ahead].GetOpCode() == OpReturn && byteutil.ToHex(insts[ahead].GetLeft().Bytes()) == opened {
			closes, answer = ahead, insts[ahead].GetRight()
			break
		}
	}

	rest := b.reserve()
	inside := b.reserve()
	b.run(inside, insts[at+1:closes], 0, func() Terminator { return Goes(rest, answer) })
	b.run(rest, insts[min(closes+1, len(insts)):], 1, tail)

	b.put(id, params, held, Goes(inside), origin)
}

// branch splits a run at an "if" into the two arms and the block they meet at.
//
// The counts say where each piece is: the arm taken when the test holds is the instructions
// the "if" says to skip, the other reaches to where the jump closing the first one lands, and
// what is left is where both arrive. Those counts are read here and nowhere after — what comes
// out of this names blocks.
func (b *blocking) branch(id BlockID, params int, held []Instruction, origin Origin, insts []Instruction, at int, tail ends) {
	inst := insts[at]
	otherwise := at + 1 + int(byteutil.ToUint64(inst.GetRight().Bytes()))

	// The arm taken when the test holds ends by jumping over the other one, and that jump is
	// what says where the two meet.
	meeting := len(insts)
	if jump := otherwise - 1; jump >= 0 && jump < len(insts) && insts[jump].GetOpCode() == OpJump {
		meeting = min(jump+1+int(byteutil.ToUint64(insts[jump].GetLeft().Bytes())), len(insts))
	}

	// The block both arms arrive at is named before either of them is read, which is the whole
	// reason a block is reserved apart from being filled.
	meet := b.reserve()
	whenTrue := b.arm(insts[at+1:otherwise], meet)
	whenFalse := b.arm(insts[min(otherwise, meeting):meeting], meet)
	// What the run this branch was part of does when it reaches its end is what the block the
	// arms meet at does: the branch is inside that run, not around it. A branch written inside
	// an arm of another meets, and then carries on to where the outer arms meet — and getting
	// that wrong ended the outer scope in the middle of it.
	b.run(meet, insts[meeting:], 1, tail)

	b.put(id, params, held, Chooses(inst.GetLeft(), To(whenTrue), To(whenFalse)), origin)
}

// arm turns one side of a branch into a block that hands its value to the block the two meet
// at.
//
// What it answers with is its own closing return — the one naming the "if" rather than a
// scope, which used to mean "the value of this arm". It is taken off the end rather than
// searched for, because a branch written inside this one has returns of its own and they
// belong to it.
//
// Both arms answering with one value is what makes an "if" an expression. Saying it here, as a
// value handed over, is what turns that from a convention two consumers each had to keep into
// something written down.
func (b *blocking) arm(insts []Instruction, meet BlockID) BlockID {
	answer := Nothing()

	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == OpJump {
		insts = insts[:len(insts)-1]
	}
	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == OpReturn {
		answer = insts[len(insts)-1].GetRight()
		insts = insts[:len(insts)-1]
	}

	id := b.reserve()
	b.run(id, insts, 0, func() Terminator { return Goes(meet, answer) })
	return id
}

// opened answers a scope's body without the instruction that opened it. The block is the
// scope, so what opened it has nothing left to do — and leaving it in would make the body look
// like a block written inside itself.
func opened(body []Instruction) []Instruction {
	if len(body) > 0 && body[0].GetOpCode() == OpBeginScope {
		return body[1:]
	}
	return body
}

// feedsRead answers how many positions a body addresses. A scope written inside it reads its
// own vector, so its feeds say nothing about this one.
func feedsRead(insts []Instruction) int {
	highest := -1
	for at := 0; at < len(insts); at++ {
		inst := insts[at]
		if inst.GetOpCode() == OpDefer {
			at += int(byteutil.ToUint64(inst.GetRight().Bytes()))
			continue
		}
		if inst.GetOpCode() != OpGetFeed {
			continue
		}
		if read := int(byteutil.ToUint64(inst.GetLeft().Bytes())); read > highest {
			highest = read
		}
	}
	return highest + 1
}

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
