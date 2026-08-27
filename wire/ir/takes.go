package ir

import "fmt"

// What each opcode takes, written where the opcode is written.
//
// This is the second half of the disease the IR was reshaped to cure. The first made a
// consumer guess structure; this one made it guess meaning — the same bytes are a number under
// one kind and a name under another, and which one an opcode wants was a comment beside it and
// a decision in every consumer's head.
//
// It has cost twice, both times the same way: somebody found out afterwards and patched the
// side that happened to be right. A comment cannot be checked. This can.

// A Slot says what one operand may be.
//
// It is a class of kinds rather than one, because more than one kind carries the same idea. A
// value is a value whether the program computed it — a Ref — or wrote it down, an Imm; a
// consumer that cared about the difference would be a consumer for whom folding a literal
// changed the program.
type Slot byte

const (
	// AValue is something of the program: computed by another instruction, or written down.
	AValue Slot = iota
	// AName is something that outlives the instruction: a name a scope bound, or a scope to
	// call.
	AName
	// ANumber is a number the operation takes about itself — a width, an index, a position.
	// It is Const where the emitter says so, and Imm where the number came from the source.
	ANumber
	// AText is bytes written for a person, and never a value.
	AText
	// AnythingWritten is an operand whose kind the opcode does not constrain: a save carries
	// what it was given, and what it was given is a value or the number of a block.
	AnythingWritten
	// ANothing is an operand that is not there.
	ANothing
)

// A Shape is what an instruction takes: one Slot per operand.
//
// Repeats says the last Slot may stand for any number of operands, including none. A
// construction is what it is for — a shape has as many fields as it has — and a call, which
// applies a vector of values to a scope.
type Shape struct {
	Slots   []Slot
	Repeats bool
}

// takes is what each opcode accepts. An opcode not here is unconstrained, and Verify says
// nothing about it — which is the honest answer for one nobody has written down yet.
var takes = map[byte]Shape{
	OpMultiply:    {Slots: []Slot{AValue, AValue}},
	OpAdd:         {Slots: []Slot{AValue, AValue}},
	OpSubtract:    {Slots: []Slot{AValue, AValue}},
	OpDivide:      {Slots: []Slot{AValue, AValue}},
	OpExponential: {Slots: []Slot{AValue, AValue}},

	OpDiff:    {Slots: []Slot{AValue, AValue}},
	OpEquals:  {Slots: []Slot{AValue, AValue}},
	OpBigger:  {Slots: []Slot{AValue, AValue}},
	OpSmaller: {Slots: []Slot{AValue, AValue}},
	OpAnd:     {Slots: []Slot{AValue, AValue}},
	OpOr:      {Slots: []Slot{AValue, AValue}},

	OpIdent: {Slots: []Slot{AName, AValue}},
	// A save carries what it was handed, and that is a value of the program or the number of
	// a block — a scope bound to a name is the second. The comment beside the opcode said Imm
	// for a long time and was wrong for as long.
	OpSave: {Slots: []Slot{AnythingWritten, ANothing}},
	OpLoad: {Slots: []Slot{AName, ANothing}},

	OpGetFeed: {Slots: []Slot{ANumber, ANothing}},
	// A scope has no arity: running one is applying a vector of values to it, so the name
	// comes first and as many values as were applied follow.
	OpCall: {Slots: []Slot{AName, AValue}, Repeats: true},

	OpPrintBytes:   {Slots: []Slot{AValue, ANothing}},
	OpPrintChars:   {Slots: []Slot{AValue, ANothing}},
	OpPrintDecimal: {Slots: []Slot{AValue, ANothing}},

	OpPull: {Slots: []Slot{AValue}, Repeats: true},
	OpPush: {Slots: []Slot{AValue, AValue}},
	OpHead: {Slots: []Slot{AValue, ANumber}},
	OpTail: {Slots: []Slot{AValue, ANumber}},

	OpAssert: {Slots: []Slot{AValue, AText}},

	// A run is its tapes and nothing else, so a construction takes as many as it has.
	OpJoin:  {Slots: []Slot{AValue}, Repeats: true},
	OpField: {Slots: []Slot{AValue, ANumber, ANumber}},

	OpStorageGet: {Slots: []Slot{AValue, ANothing}},
	OpStorageSet: {Slots: []Slot{AValue, AValue}},
}

// holds says whether a kind is one of the things a slot accepts.
func (slot Slot) holds(kind Kind) bool {
	switch slot {
	case AValue:
		return kind == KindRef || kind == KindImm || kind == KindConst
	case AName:
		return kind == KindName
	case ANumber:
		return kind == KindConst || kind == KindImm
	case AText:
		return kind == KindText
	case AnythingWritten:
		return kind != KindEmpty && kind != KindTarget
	case ANothing:
		return kind == KindEmpty
	}
	return true
}

// describe says what a slot wants, in words rather than a number.
func (slot Slot) describe() string {
	switch slot {
	case AValue:
		return "a value"
	case AName:
		return "a name"
	case ANumber:
		return "a number of its own"
	case AText:
		return "text"
	case AnythingWritten:
		return "something written down"
	case ANothing:
		return "nothing"
	}
	return "anything"
}

// verifyTakes answers what is wrong with the operands of one instruction, and nothing when
// they are what its opcode takes.
func verifyTakes(block BlockID, at int, inst Instruction) []Problem {
	shape, written := takes[inst.GetOpCode()]
	if !written {
		return nil
	}

	operands := inst.GetOperands()
	found := make([]Problem, 0)

	if !shape.Repeats && len(operands) > len(shape.Slots) {
		return []Problem{{Block: block, At: at, Says: fmt.Sprintf(
			"opcode %d takes %d operands and was given %d", inst.GetOpCode(), len(shape.Slots), len(operands))}}
	}

	for position, operand := range operands {
		slot := shape.Slots[min(position, len(shape.Slots)-1)]
		if slot.holds(operand.Kind()) {
			continue
		}
		found = append(found, Problem{Block: block, At: at, Says: fmt.Sprintf(
			"operand %d of opcode %d is %s, and that operand takes %s",
			position, inst.GetOpCode(), describeKindOf(operand.Kind()), slot.describe())})
	}
	return found
}

// describeKindOf says what an operand is, in the words the kinds are documented in.
func describeKindOf(kind Kind) string {
	switch kind {
	case KindRef:
		return "a value another instruction left"
	case KindImm:
		return "a value written down"
	case KindConst:
		return "a number the operation takes about itself"
	case KindName:
		return "a name"
	case KindText:
		return "text"
	case KindBlock:
		return "the number of a block"
	case KindTarget:
		return "where control goes"
	case KindEmpty:
		return "not there"
	}
	return "something"
}
