package emitter

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

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
func BlocksOf(insts []ir.Instruction) []ir.Block {
	b := &blocking{scopes: make(map[string]ir.BlockID)}
	top := b.reserve()
	b.run(top, insts, 0, func() ir.Terminator { return ir.Ends(ir.Nothing()) })
	return b.blocks
}

// blocking builds blocks as it walks, handing each the number it will be known by.
//
// A block is reserved before it is filled, because a branch names blocks that have not been
// walked yet: the arms of an "if" are named by the terminator that chooses between them, and
// that terminator is written before either arm is read.
type blocking struct {
	blocks []ir.Block
	// scopes answers, for the label an OpDefer left, the block its body became — so the
	// binding that follows it can name a block instead of an instruction that is gone.
	scopes map[string]ir.BlockID
}

// reserve takes the next number and leaves a place for the block to be put in.
func (b *blocking) reserve() ir.BlockID {
	id := ir.BlockID(len(b.blocks))
	b.blocks = append(b.blocks, ir.Block{ID: id})
	return id
}

// put fills a block that was reserved.
func (b *blocking) put(id ir.BlockID, params int, insts []ir.Instruction, term ir.Terminator, origin ir.Origin) {
	b.blocks[id] = ir.Block{ID: id, Params: params, Insts: insts, Term: term, Origin: origin}
}

// ends says how a run finishes when it reaches the end without a return of its own.
type ends func() ir.Terminator

// run fills a reserved block from one straight run of instructions.
//
// It is one block until something splits it. A branch splits it into the run before, the two
// arms, and the block they meet at — and each of those is a straight run again, so the same
// walk handles a branch inside a branch without knowing that it is doing so.
func (b *blocking) run(id ir.BlockID, insts []ir.Instruction, params int, tail ends) {
	held := make([]ir.Instruction, 0, len(insts))
	origin := ir.Origin{}
	if len(insts) > 0 {
		origin = insts[0].GetOrigin()
	}

	for at := 0; at < len(insts); at++ {
		inst := insts[at]

		switch inst.GetOpCode() {
		case ir.OpBeginScope:
			// The block is the scope. Nothing opens it but arriving at it.

		case ir.OpDefer:
			// The body is what this instruction says is its own, and it becomes a block.
			length := int(byteutil.ToUint64(inst.GetRight().Bytes()))
			body := insts[at+1 : at+1+length]
			scope := b.reserve()
			b.run(scope, body, feedsRead(body), func() ir.Terminator { return ir.Ends(ir.Nothing()) })
			b.scopes[byteutil.ToHex(inst.GetLabel())] = scope
			at += length

		case ir.OpIdent:
			// A binding whose value was a scope names the block instead of an instruction
			// that is no longer there.
			if scope, ok := b.scopes[byteutil.ToHex(inst.GetRight().Bytes())]; ok {
				inst = ir.NewInstruction(
					inst.GetLabel(), ir.OpIdent, inst.GetLeft(), ir.BlockOf(scope)).At(inst.GetOrigin())
			}
			held = append(held, inst)

		case ir.OpReturn:
			// A scope ends here, and answers with what the return names.
			b.put(id, params, held, ir.Ends(inst.GetRight()), origin)
			return

		case ir.OpIf:
			b.branch(id, params, held, origin, insts, at)
			return

		default:
			held = append(held, inst)
		}
	}

	b.put(id, params, held, tail(), origin)
}

// branch splits a run at an "if" into the two arms and the block they meet at.
//
// The counts say where each piece is: the arm taken when the test holds is the instructions
// the "if" says to skip, the other reaches to where the jump closing the first one lands, and
// what is left is where both arrive. Those counts are read here and nowhere after — what comes
// out of this names blocks.
func (b *blocking) branch(id ir.BlockID, params int, held []ir.Instruction, origin ir.Origin, insts []ir.Instruction, at int) {
	inst := insts[at]
	otherwise := at + 1 + int(byteutil.ToUint64(inst.GetRight().Bytes()))

	// The arm taken when the test holds ends by jumping over the other one, and that jump is
	// what says where the two meet.
	meeting := len(insts)
	if jump := otherwise - 1; jump >= 0 && jump < len(insts) && insts[jump].GetOpCode() == ir.OpJump {
		meeting = min(jump+1+int(byteutil.ToUint64(insts[jump].GetLeft().Bytes())), len(insts))
	}

	// The block both arms arrive at is named before either of them is read, which is the whole
	// reason a block is reserved apart from being filled.
	meet := b.reserve()
	whenTrue := b.arm(insts[at+1:otherwise], meet)
	whenFalse := b.arm(insts[min(otherwise, meeting):meeting], meet)
	b.run(meet, insts[meeting:], 1, func() ir.Terminator { return ir.Ends(ir.Nothing()) })

	b.put(id, params, held, ir.Chooses(inst.GetLeft(), ir.To(whenTrue), ir.To(whenFalse)), origin)
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
func (b *blocking) arm(insts []ir.Instruction, meet ir.BlockID) ir.BlockID {
	answer := ir.Nothing()

	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == ir.OpJump {
		insts = insts[:len(insts)-1]
	}
	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == ir.OpReturn {
		answer = insts[len(insts)-1].GetRight()
		insts = insts[:len(insts)-1]
	}

	id := b.reserve()
	b.run(id, insts, 0, func() ir.Terminator { return ir.Goes(meet, answer) })
	return id
}

// feedsRead answers how many positions a body addresses. A scope written inside it reads its
// own vector, so its feeds say nothing about this one.
func feedsRead(insts []ir.Instruction) int {
	highest := -1
	for at := 0; at < len(insts); at++ {
		inst := insts[at]
		if inst.GetOpCode() == ir.OpDefer {
			at += int(byteutil.ToUint64(inst.GetRight().Bytes()))
			continue
		}
		if inst.GetOpCode() != ir.OpGetFeed {
			continue
		}
		if read := int(byteutil.ToUint64(inst.GetLeft().Bytes())); read > highest {
			highest = read
		}
	}
	return highest + 1
}
