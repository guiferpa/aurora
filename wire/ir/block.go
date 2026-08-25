package ir

// A BlockID names a block. It is a number and not a position: the whole point of a block is
// that where it sits in a list says nothing about it.
type BlockID int

// A Point is a place in a program: a block, and how far into it.
//
// It is what an offset into a list of instructions used to be, and it needs two numbers for
// the reason the whole of this does: instructions that run one after another are not
// necessarily written one after another, so where something is cannot be counted to.
type Point struct {
	Block BlockID
	At    int
}

// A Block is a run of instructions with one way in and one way out.
//
// It is what the instruction list does not say. Today a scope is found by reading a length off
// OpDefer and slicing, and a branch by reading a count off OpIf and adding — so the structure
// of a program lives in the order of the list and in arithmetic over indices, and every
// consumer works it out again, each in its own way. That is not a theoretical cost: the
// builder found scopes by counting only at the top of the list, so a scope written inside
// another escaped it and was written as if it were straight-line code.
//
// A block says it instead. Control arrives at the first instruction and leaves at the
// terminator, and nowhere else — which is what makes every instruction in it movable, and what
// lets a jump name where it goes rather than count how far.
type Block struct {
	ID BlockID
	// Params is what arrives with control, and what each of them is known as inside.
	//
	// A scope's are unnamed: they arrive as the vector applied to it, and feed reads a
	// position of that vector rather than a name. A block reached from somewhere that hands a
	// value over names the one it takes — where the arms of a branch meet, and where a block
	// written inside an expression carries on — because whoever reads that value reads it
	// under the name the value had before the split.
	//
	// Naming it is what makes the handover checkable. It used to be an agreement: off a chain
	// a name in a map, on one a place on the stack, and nothing in the IR either way.
	Params []Label
	Insts  []Instruction
	Term   Terminator
	Origin Origin
}

// A TermKind is how a block ends. There are three, and there is no fourth: a block either
// answers, goes somewhere, or chooses between two somewheres.
type TermKind byte

const (
	// Ret ends the block and answers with Value. For a scope's last block that is the scope's
	// answer.
	Ret TermKind = iota
	// Br goes to the one block in Targets.
	Br
	// BrIf goes to the first block in Targets when Cond is not the neutral tape, and to the
	// second when it is.
	BrIf
)

// A Terminator is how a block ends, and the only thing in the IR that decides where control
// goes.
//
// Keeping it out of the instructions is the point. An instruction computes a value from its
// operands and does nothing else, so a consumer may hold it back, move it next to whoever
// takes it, or drop it when nobody does — none of which is safe for something that can send
// control somewhere. The builder already learned this the hard way and has a list of opcodes
// it refuses to move across; with a terminator there is nothing to list.
type Terminator struct {
	Kind TermKind
	// Cond is what BrIf chooses on.
	Cond Operand
	// Value is what Ret answers with.
	Value Operand
	// Targets is where control goes: one block for Br, two for BrIf, none for Ret.
	Targets []Target
}

// A Target is a block and the values handed to its parameters.
//
// The arms of a branch meet by each handing its value to the block they meet at, rather than
// by agreeing on a place to leave it. Off a chain that place was a name in a map; on one it
// was a position on the stack. Neither is in the IR, so neither could be checked — and a
// consumer that got it wrong got it wrong quietly.
type Target struct {
	Block BlockID
	Args  []Operand
}

// Ends answers a terminator that ends a block with a value.
func Ends(value Operand) Terminator {
	return Terminator{Kind: Ret, Value: value}
}

// Goes answers a terminator that hands control, and values, to one block.
func Goes(to BlockID, args ...Operand) Terminator {
	return Terminator{Kind: Br, Targets: []Target{{Block: to, Args: args}}}
}

// Chooses answers a terminator that goes one way when the condition holds and the other when
// it does not. Both arms are named here, so neither is "the instructions that follow".
func Chooses(cond Operand, whenTrue, whenFalse Target) Terminator {
	return Terminator{Kind: BrIf, Cond: cond, Targets: []Target{whenTrue, whenFalse}}
}

// To answers a target: a block, and what is handed to its parameters.
func To(block BlockID, args ...Operand) Target {
	return Target{Block: block, Args: args}
}
