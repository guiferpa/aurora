package evm

import (
	"io"

	"github.com/guiferpa/aurora/wire/ir"
)

// layoutOf answers the blocks of one scope, in the order they are written.
//
// The blocks of a scope are the ones control can reach from its first, and a terminator is the
// only thing that moves control — so reachability is the whole answer, and a scope written
// inside another is not in it. That used to be worked out by reading a length and slicing, and
// only at the top of the instruction list, which is how a scope written inside another came to
// be written as if it were straight-line code.
//
// The order is reverse postorder with the ways out walked backwards, which puts a block before
// everywhere it goes and lays a branch out as the test, the arm taken when it holds, the other
// arm, and then where the two meet. That is the order a branch was written in before there
// were blocks, and it is the one that leaves the most jumps as simply carrying on.
func layoutOf(blocks []ir.Block, entry ir.BlockID) []ir.BlockID {
	seen := make(map[ir.BlockID]bool, len(blocks))
	order := make([]ir.BlockID, 0, len(blocks))

	var walk func(id ir.BlockID)
	walk = func(id ir.BlockID) {
		if seen[id] || int(id) >= len(blocks) {
			return
		}
		seen[id] = true
		targets := blocks[id].Term.Targets
		for at := len(targets) - 1; at >= 0; at-- {
			walk(targets[at].Block)
		}
		order = append(order, id)
	}
	walk(entry)

	for left, right := 0, len(order)-1; left < right; left, right = left+1, right-1 {
		order[left], order[right] = order[right], order[left]
	}
	return order
}

// writeBlocks writes the blocks of one scope, and answers where each of them starts.
//
// A block that is gone to needs somewhere to arrive at, and the EVM refuses a jump to any byte
// that is not a JUMPDEST — so every block opens with one. It costs a byte and a gas, and it is
// the price of a program having more than one shape.
//
// The addresses are found by writing to something that only counts, which is how every address
// in this backend is found: a table of sizes beside the writer is a second description of the
// same bytes, and two descriptions drift.
func writeBlocks(bs io.Writer, blocks []ir.Block, order []ir.BlockID, base int, scope Scope) error {
	at := make(map[ir.BlockID]int, len(order))
	offset := base
	for _, id := range order {
		at[id] = offset
		var measured counter
		if err := writeBlock(&measured, blocks[id], order, at, scope, id); err != nil {
			return err
		}
		offset += int(measured)
	}

	for _, id := range order {
		if err := writeBlock(bs, blocks[id], order, at, scope, id); err != nil {
			return err
		}
	}
	return nil
}

// A cursor writes through to another writer and keeps where it is.
//
// One instruction needs to know the byte it starts at: a call carries its own landing, and
// works the address out from where it began. Nothing else does, and nothing else has to — an
// instruction that does not move control does not care where it is.
type cursor struct {
	to io.Writer
	at int
}

func (c *cursor) Write(p []byte) (int, error) {
	n, err := c.to.Write(p)
	c.at += n
	return n, err
}

// writeBlock writes one block: somewhere to arrive at, what it computes, and how it ends.
func writeBlock(bs io.Writer, block ir.Block, order []ir.BlockID, at map[ir.BlockID]int, scope Scope, id ir.BlockID) error {
	here := &cursor{to: bs, at: at[id]}
	if _, err := here.Write([]byte{OpJumpDestiny}); err != nil {
		return err
	}

	for _, inst := range block.Insts {
		if err := WriteInstruction(here, scope.Names, inst, scope, here.at); err != nil {
			return err
		}
	}

	return writeTerminator(here, block.Term, order, at, scope, id)
}

// writeTerminator writes how a block ends.
//
// Going somewhere that is written next is carrying on, and costs nothing. Everything else is a
// jump, and a jump carries the address of a block rather than a count of instructions — which
// is the difference the blocks were for: nothing here has to know how long anything is.
func writeTerminator(bs io.Writer, term ir.Terminator, order []ir.BlockID, at map[ir.BlockID]int, scope Scope, id ir.BlockID) error {
	switch term.Kind {
	case ir.Ret:
		// The value is on the stack, left there by whoever computed it — unless the program
		// wrote it down, or there is none, in which case it reaches the stack here.
		if term.Value.Kind() == ir.KindImm {
			if _, err := WritePush(bs, term.Value.Bytes(), scope.TapeSize); err != nil {
				return err
			}
		}
		if term.Value.Kind() == ir.KindEmpty {
			if _, err := WritePush(bs, nil, scope.TapeSize); err != nil {
				return err
			}
		}
		write := WriteReturnToCaller
		if scope.Answers {
			write = WriteReturnToChain
		}
		_, err := write(bs)
		return err

	case ir.Br:
		return writeGoes(bs, term.Targets[0], order, at, id)

	case ir.BrIf:
		// The IR goes the second way when the test does not hold, and the EVM jumps when what
		// it pops is not zero, so the test is turned over first.
		if _, err := bs.Write([]byte{OpIsZero}); err != nil {
			return err
		}
		if _, err := WritePush2(bs, at[term.Targets[1].Block]); err != nil {
			return err
		}
		if _, err := bs.Write([]byte{OpJumpIf}); err != nil {
			return err
		}
		return writeGoes(bs, term.Targets[0], order, at, id)
	}

	return nil
}

// writeGoes hands control to a block, and carries on instead when that block is the one
// written next.
func writeGoes(bs io.Writer, target ir.Target, order []ir.BlockID, at map[ir.BlockID]int, id ir.BlockID) error {
	for pos, written := range order {
		if written == id && pos+1 < len(order) && order[pos+1] == target.Block {
			return nil
		}
	}
	if _, err := WritePush2(bs, at[target.Block]); err != nil {
		return err
	}
	_, err := bs.Write([]byte{OpJump})
	return err
}
