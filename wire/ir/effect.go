package ir

// An Effect says what an instruction does beside leaving a value.
//
// A consumer may hold an instruction back and emit it next to whoever takes its value, which
// is sound only when the value it leaves is the whole of what it does. The builder does
// exactly that, and what kept it honest until now were three lists of opcodes in
// builder/evm — and a list of opcodes is how this project has been wrong before: whoever
// writes the next opcode has to remember every one of them, and nothing reminds them.
//
// This says it once, where the opcode is written down.
type Effect byte

const (
	// Pure is an instruction whose value is all it does. It may be moved anywhere its
	// operands allow.
	Pure Effect = iota
	// Reads depends on state something else can change: a name in the frame today, a slot
	// of a contract's storage when that arrives.
	Reads
	// Writes changes that state, or says something whose order matters.
	Writes
	// Escapes leaves the program: another contract runs, and it may do anything, including
	// coming back in. Nothing crosses it.
	//
	// Nothing is Escapes yet. It is named here because the set is fixed with what is coming
	// in view, so a backend refusing what it cannot carry can say which of the four it met
	// rather than which opcode.
	Escapes
)

// MayCross answers whether two instructions can be put in the other order.
//
// Two instructions swap unless one of them changes something the other would notice. That is
// the whole rule, and the sixteen pairs come out of it rather than being listed: a Pure
// changes nothing and notices nothing, so it crosses anything; two reads of the frame commute,
// because neither of them moves; and nothing at all crosses a write.
//
// It started stricter — swap only if one of them is Pure — and a measurement over real
// programs said that was too strict. `a - b` puts the two loads on the stack in the order the
// subtraction needs them, which swaps two reads, which is safe and which the strict rule
// refused. Deriving the pairs from what an instruction does costs four lines more than the
// strict version and answers every cell, including the ones nothing reaches yet.
func MayCross(a, b Effect) bool {
	return !disturbs(a, b) && !disturbs(b, a)
}

// disturbs says whether the first changes something the second would notice, which is the
// whole of why two instructions have to stay in the order they were written.
func disturbs(a, b Effect) bool {
	return changes(a) && notices(b)
}

// changes says whether an instruction leaves anything behind but its value.
func changes(e Effect) bool {
	return e == Writes || e == Escapes
}

// notices says whether an instruction can tell that something else changed. A write notices
// too: two writes to the same place in the other order is a different program.
func notices(e Effect) bool {
	return e != Pure
}

// effects is what each opcode does. An opcode that is not here is Pure, which is most of
// them: arithmetic, comparison, logic and the tape operations all compute and nothing else.
var effects = map[byte]Effect{
	// The frame is memory, and a binding writes it while a load reads it. Nothing moves them
	// wrongly today, because the one thing that moves instructions moves them to put operands
	// in order — but the frame is state, and the rule that holds storage holds this too.
	OpIdent: Writes,
	OpLoad:  Reads,

	// Storage is state in the plainest sense: it outlives the transaction, and two of these in
	// the other order is a different program. They are what the rule was written for, before
	// there was anything for it to hold.
	OpStorageGet: Reads,
	OpStorageSet: Writes,

	// What a transaction carried is not here, and Pure is right rather than lenient: it is
	// fixed for the whole of one call, so two reads of it in either order are the same two
	// reads. Nothing in the program can change it, and nothing outside can change it while
	// the program runs.

	// A print says something, and when it says it is the whole of what it is for. It never
	// reaches the backend that reorders — a chain has nowhere to put a log, by decision — so
	// this costs nothing and is still the truth.
	OpPrintBytes:   Writes,
	OpPrintChars:   Writes,
	OpPrintDecimal: Writes,
}

// keeps names the instructions whose work outlives the program that did it.
//
// It is a different question from the effect, and reading them as one is a mistake worth
// naming here so nobody makes it twice. An effect answers whether two instructions may swap,
// and by that measure a binding writes: the frame is memory, and a load after a store must
// stay after it. But a frame is gone when the scope is, and nothing outside the program ever
// saw it — so a scope that binds a name changes nothing anybody can come back and read.
//
// A print is the sharper case: its effect is Writes, because when it says something matters,
// and it reaches no bytecode at all. On a chain it is the identity.
//
// So this is the short list, and it is short because there is one thing a program can do today
// that survives it. Events and an external call join it when they are written.
var keeps = map[byte]bool{
	OpStorageSet: true,
}

// Keeps says whether what an instruction does outlives the program that did it.
//
// It is what tells a question from a change: a scope that only computes can be asked, against
// the state as it is, for nothing; a scope that keeps something has to be sent.
func Keeps(op byte) bool {
	return keeps[op]
}

// EffectOf answers what an opcode does beside leaving a value.
//
// It is a function of the opcode rather than a field on the instruction, which is where the
// IR RFC first put it. A field is set by whoever builds the instruction, so it can be set
// wrong and nothing notices; an opcode has one effect wherever it is written, and saying it
// once is both fewer lines to read and the only version that cannot disagree with itself.
// The day an instruction's effect depends on more than its opcode, the field arrives then and
// has a reason.
func EffectOf(op byte) Effect {
	return effects[op]
}
