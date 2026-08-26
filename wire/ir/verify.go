package ir

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// Whether a program says what it means, asked before any consumer touches it.
//
// This is what the rest of the IR was for. A consumer used to be handed a list and left to
// work out what it meant — where a scope began, which operand named a value, how far a branch
// jumped — and the failure was always the same shape: the binary came out, deployed, answered,
// and was quietly wrong. What kept that from happening was a list of opcodes in the backend,
// kept in agreement with the emitter by somebody remembering.
//
// A check is not a list. It is the difference between a compiler that refuses and one that
// hands over bytes nobody asked for.

// A Problem is one thing wrong with a program, and where it is.
//
// Where matters more than it looks. A block and a position are what an origin is worked out
// from, and a problem nobody can find is a problem nobody fixes.
type Problem struct {
	Block BlockID
	// At is which instruction, or WhereItEnds when the terminator is what is wrong.
	At   int
	Says string
}

// WhereItEnds is the position of the thing that ends a block, which is not one of its
// instructions and so has no index among them.
const WhereItEnds = -1

func (p Problem) Error() string {
	if p.At == WhereItEnds {
		return fmt.Sprintf("block %d, where it ends: %s", p.Block, p.Says)
	}
	return fmt.Sprintf("block %d, instruction %d: %s", p.Block, p.At, p.Says)
}

// Verify answers everything wrong with a program, and nothing when it is sound.
//
// It answers all of them rather than the first, because a program that is wrong is usually
// wrong in several places and stopping at one turns fixing it into a loop. A consumer that
// wants one takes the first.
func Verify(blocks []Block) []Problem {
	found := make([]Problem, 0)
	for _, block := range blocks {
		found = append(found, verifyBlock(blocks, block)...)
	}
	return found
}

// verifyBlock asks of one block everything that can be asked of it alone.
func verifyBlock(blocks []Block, block Block) []Problem {
	found := make([]Problem, 0)

	// A value is defined where the instruction that leaves it is written, and a label is
	// unique in its block — so a label written twice is two values under one name, and every
	// reader after it means whichever the reader happened to reach.
	defined := make(map[string]bool, len(block.Params)+len(block.Insts))
	for _, param := range block.Params {
		defined[byteutil.ToHex(param)] = true
	}

	for at, inst := range block.Insts {
		// Read before written: a Ref names a value that already exists, which is what makes
		// the order of a block mean something.
		for _, operand := range inst.GetOperands() {
			if operand.Kind() != KindRef {
				continue
			}
			if !defined[byteutil.ToHex(operand.Bytes())] {
				found = append(found, Problem{Block: block.ID, At: at,
					Says: fmt.Sprintf("it reads %s, and nothing before it leaves a value under that name", byteutil.ToHex(operand.Bytes()))})
			}
		}

		label := byteutil.ToHex(inst.GetLabel())
		if defined[label] {
			found = append(found, Problem{Block: block.ID, At: at,
				Says: fmt.Sprintf("it leaves a value under %s, and so does something before it", label)})
		}
		defined[label] = true
	}

	return append(found, verifyTerminator(blocks, block, defined)...)
}

// verifyTerminator asks of the way a block ends the three things that can be wrong with it.
func verifyTerminator(blocks []Block, block Block, defined map[string]bool) []Problem {
	found := make([]Problem, 0)
	at := Problem{Block: block.ID, At: WhereItEnds}

	if wanted, given := targetsWanted(block.Term.Kind), len(block.Term.Targets); wanted != given {
		at.Says = fmt.Sprintf("it %s and names %d blocks, and that way of ending names %d",
			describeKind(block.Term.Kind), given, wanted)
		found = append(found, at)
	}

	for _, operand := range []Operand{block.Term.Cond, block.Term.Value} {
		if operand.Kind() != KindRef || defined[byteutil.ToHex(operand.Bytes())] {
			continue
		}
		found = append(found, Problem{Block: block.ID, At: WhereItEnds,
			Says: fmt.Sprintf("it reads %s, and nothing in the block leaves a value under that name", byteutil.ToHex(operand.Bytes()))})
	}

	for _, target := range block.Term.Targets {
		found = append(found, verifyTarget(blocks, block, target, defined)...)
	}
	return found
}

// verifyTarget asks of one place control goes whether it exists and whether it is handed what
// it takes.
//
// A branch is checked for how many values it hands over and a call is not, and that is the
// language rather than an omission: a scope has no arity — running one is applying a vector of
// values to it, and feed reads a position of that vector — so there is nothing to count.
func verifyTarget(blocks []Block, block Block, target Target, defined map[string]bool) []Problem {
	at := Problem{Block: block.ID, At: WhereItEnds}

	if target.Block < 0 || int(target.Block) >= len(blocks) {
		at.Says = fmt.Sprintf("it goes to block %d, and the program has %d", target.Block, len(blocks))
		return []Problem{at}
	}

	found := make([]Problem, 0)
	if wanted := len(blocks[target.Block].Params); len(target.Args) != wanted {
		at.Says = fmt.Sprintf("it hands %d values to block %d, which takes %d",
			len(target.Args), target.Block, wanted)
		found = append(found, at)
	}
	for _, arg := range target.Args {
		if arg.Kind() != KindRef || defined[byteutil.ToHex(arg.Bytes())] {
			continue
		}
		found = append(found, Problem{Block: block.ID, At: WhereItEnds,
			Says: fmt.Sprintf("it hands over %s, and nothing in the block leaves a value under that name", byteutil.ToHex(arg.Bytes()))})
	}
	return found
}

// targetsWanted says how many blocks each way of ending names. There are three, and there is
// no fourth.
func targetsWanted(kind TermKind) int {
	switch kind {
	case Br:
		return 1
	case BrIf:
		return 2
	default:
		return 0
	}
}

// describeKind says how a block ends, in the words the language uses rather than a number.
func describeKind(kind TermKind) string {
	switch kind {
	case Br:
		return "goes somewhere"
	case BrIf:
		return "chooses between two somewheres"
	default:
		return "returns"
	}
}
