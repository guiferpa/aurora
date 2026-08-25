package emitter

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// The opcodes the emitter uses to write structure down on its way to the blocks.
//
// They are the emitter's own and never leave it: what a phase after this one is handed is
// blocks, where the structure is the block and its terminator. They used to be opcodes of the
// IR, which meant every consumer could see them and had to decide what they meant — and each
// decided by counting, in its own way.
//
// The numbers are its own too, above anything the IR declares, so nothing can be mistaken for
// one of them.
const (
	opBeginScope = 0xf0 + iota // opens a scope, and leaves the value its body ends with
	opDefer                    // Ref, Target -> the scope that follows, and how long its body is
	opIf                       // Ref, Target -> skips ahead when the test is false
	opJump                     // Target -> skips ahead, always
	opReturn                   // Ref, Ref -> the value of the scope, or of the arm of an if
)

// blocksOf answers the same program as blocks with terminators.
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
// instruction: opDefer, opBeginScope, opIf, opJump and opReturn. What is left inside a block
// computes values and nothing else.
func blocksOf(insts []ir.Instruction) []ir.Block {
	blocks, _ := placedBlocksOf(insts)
	return blocks
}

// PlacedBlocksOf answers the same blocks, and where each instruction of the list ended up.
//
// Whoever compiled the list knows things about it in terms of the list — which stretch of it
// each top-level expression is, so a runner can stop between them and answer where each one
// happens rather than all of them at the end. Those stretches have to move with the
// instructions, and an instruction's place is now two numbers rather than one.
func placedBlocksOf(insts []ir.Instruction) ([]ir.Block, map[string]ir.Point) {
	b := &blocking{places: make(map[string]ir.Point)}
	top := b.reserve()
	// The top of a program takes values the way a scope does. ir.Nothing applies any to it — no
	// transaction names it — so what it reads there is zeros, which is what reading past what
	// was applied answers. Saying how many it addresses is still what keeps the names it binds
	// from being kept in the same places.
	b.run(top, insts, applied(feedsRead(insts)), func() ir.Terminator { return ir.Ends(ir.Nothing()) })
	return b.blocks, b.places
}

// blocking builds blocks as it walks, handing each the number it will be known by.
//
// A block is reserved before it is filled, because a branch names blocks that have not been
// walked yet: the arms of an "if" are named by the terminator that chooses between them, and
// that terminator is written before either arm is read.
type blocking struct {
	blocks []ir.Block
	// places answers where each instruction ended up, by the label it leaves its value under.
	places map[string]ir.Point
}

// reserve takes the next number and leaves a place for the block to be put in.
func (b *blocking) reserve() ir.BlockID {
	id := ir.BlockID(len(b.blocks))
	b.blocks = append(b.blocks, ir.Block{ID: id})
	return id
}

// put fills a block that was reserved, and writes down where each of its instructions landed.
func (b *blocking) put(id ir.BlockID, params []ir.Label, insts []ir.Instruction, term ir.Terminator, origin ir.Origin) {
	b.blocks[id] = ir.Block{ID: id, Params: params, Insts: insts, Term: term, Origin: origin}
	for at, inst := range insts {
		b.places[byteutil.ToHex(inst.GetLabel())] = ir.Point{Block: id, At: at}
	}
}

// ends says how a run finishes when it reaches the end without a return of its own.
type ends func() ir.Terminator

// run fills a reserved block from one straight run of instructions.
//
// It is one block until something splits it. A branch splits it into the run before, the two
// arms, and the block they meet at — and each of those is a straight run again, so the same
// walk handles a branch inside a branch without knowing that it is doing so.
func (b *blocking) run(id ir.BlockID, insts []ir.Instruction, params []ir.Label, tail ends) {
	held := make([]ir.Instruction, 0, len(insts))
	origin := ir.Origin{}
	if len(insts) > 0 {
		origin = insts[0].GetOrigin()
	}

	for at := 0; at < len(insts); at++ {
		inst := insts[at]

		switch inst.GetOpCode() {
		case opBeginScope:
			// Whatever opened this run was taken off before it got here, so anything that
			// opens now is a block written inside it.
			b.inline(id, params, held, origin, insts, at, tail)
			return

		case opDefer:
			// The body is what this instruction says is its own, and it becomes a block.
			length := int(byteutil.ToUint64(inst.GetRight().Bytes()))
			body := insts[at+1 : at+1+length]
			scope := b.reserve()
			b.run(scope, opened(body), applied(feedsRead(body)), func() ir.Terminator { return ir.Ends(ir.Nothing()) })

			// A scope is a value like any other, and what it is worth is the block its body
			// became. Saying it as an ordinary instruction is what lets a scope nobody bound
			// to a name still be worth something, and lets the binding that usually follows
			// be an ordinary binding rather than a case of its own.
			held = append(held, ir.NewInstruction(inst.GetLabel(), ir.OpSave, ir.BlockOf(scope), ir.Nothing()))
			at += length

		case opReturn:
			// Whatever this closes was taken off before it got here, so a return that
			// arrives is one nothing claimed: the run ends, answering with what it names.
			b.put(id, params, held, ir.Ends(inst.GetRight()), origin)
			return

		case opIf:
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
func (b *blocking) inline(id ir.BlockID, params []ir.Label, held []ir.Instruction, origin ir.Origin, insts []ir.Instruction, at int, tail ends) {
	opened := byteutil.ToHex(insts[at].GetLabel())

	closes := len(insts)
	answer := ir.Nothing()
	for ahead := at + 1; ahead < len(insts); ahead++ {
		if insts[ahead].GetOpCode() == opReturn && byteutil.ToHex(insts[ahead].GetLeft().Bytes()) == opened {
			closes, answer = ahead, insts[ahead].GetRight()
			break
		}
	}

	rest := b.reserve()
	inside := b.reserve()
	b.run(inside, insts[at+1:closes], nil, func() ir.Terminator { return ir.Goes(rest, answer) })
	b.run(rest, insts[min(closes+1, len(insts)):], []ir.Label{insts[at].GetLabel()}, tail)

	b.put(id, params, held, ir.Goes(inside), origin)
}

// branch splits a run at an "if" into the two arms and the block they meet at.
//
// The counts say where each piece is: the arm taken when the test holds is the instructions
// the "if" says to skip, the other reaches to where the jump closing the first one lands, and
// what is left is where both arrive. Those counts are read here and nowhere after — what comes
// out of this names blocks.
func (b *blocking) branch(id ir.BlockID, params []ir.Label, held []ir.Instruction, origin ir.Origin, insts []ir.Instruction, at int, tail ends) {
	inst := insts[at]
	otherwise := at + 1 + int(byteutil.ToUint64(inst.GetRight().Bytes()))

	// The arm taken when the test holds ends by jumping over the other one, and that jump is
	// what says where the two meet.
	meeting := len(insts)
	if jump := otherwise - 1; jump >= 0 && jump < len(insts) && insts[jump].GetOpCode() == opJump {
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
	b.run(meet, insts[meeting:], []ir.Label{inst.GetLabel()}, tail)

	b.put(id, params, held, ir.Chooses(inst.GetLeft(), ir.To(whenTrue), ir.To(whenFalse)), origin)
}

// arm turns one side of a branch into a block that hands its value to the block the two meet
// at.
//
// What it returns is its own closing return — the one naming the "if" rather than a
// scope, which used to mean "the value of this arm". It is taken off the end rather than
// searched for, because a branch written inside this one has returns of its own and they
// belong to it.
//
// Both arms answering with one value is what makes an "if" an expression. Saying it here, as a
// value handed over, is what turns that from a convention two consumers each had to keep into
// something written down.
func (b *blocking) arm(insts []ir.Instruction, meet ir.BlockID) ir.BlockID {
	answer := ir.Nothing()

	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == opJump {
		insts = insts[:len(insts)-1]
	}
	if len(insts) > 0 && insts[len(insts)-1].GetOpCode() == opReturn {
		answer = insts[len(insts)-1].GetRight()
		insts = insts[:len(insts)-1]
	}

	id := b.reserve()
	b.run(id, insts, nil, func() ir.Terminator { return ir.Goes(meet, answer) })
	return id
}

// applied answers the parameters of a scope: as many as its body reads, and none of them
// named. They arrive as the vector applied to the scope, and feed reads a position of it.
func applied(reads int) []ir.Label {
	if reads == 0 {
		return nil
	}
	return make([]ir.Label, reads)
}

// opened answers a scope's body without the instruction that opened it. The block is the
// scope, so what opened it has nothing left to do — and leaving it in would make the body look
// like a block written inside itself.
func opened(body []ir.Instruction) []ir.Instruction {
	if len(body) > 0 && body[0].GetOpCode() == opBeginScope {
		return body[1:]
	}
	return body
}

// feedsRead answers how many positions a body addresses. A scope written inside it reads its
// own vector, so its feeds say nothing about this one.
func feedsRead(insts []ir.Instruction) int {
	highest := -1
	for at := 0; at < len(insts); at++ {
		inst := insts[at]
		if inst.GetOpCode() == opDefer {
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
