package ir

import "github.com/guiferpa/aurora/byteutil"

// Kind says how the bytes of an operand are read.
//
// An instruction points at two things, and nothing in a slice of bytes says what either one
// is: the same 0x03 is the number three under Imm, the value labelled "03" under Ref, and
// three instructions ahead under Target. Whoever consumed the IR had to know that per opcode
// and remember it — a table kept in agreement with the emitter by hand, which is the kind of
// agreement that quietly stops holding.
type Kind byte

const (
	// Empty is an instruction that points at one thing, or at none. It is not the same as
	// an operand carrying no bytes: OpSave of an empty run points at something, and the
	// right of OpSave points at nothing at all.
	Empty Kind = iota

	// Ref is a value another instruction left behind, named by the label that instruction
	// carries. It is the whole reason the IR is not a stack machine: a value is named where
	// it is produced and cited where it is used, and the distance between the two is a
	// question for whoever consumes this — a register allocator on one target, a stack
	// scheduler on another.
	Ref

	// Imm is the bytes themselves. There is nothing to look up: a tape a literal compiled
	// to, the width of a slice, the index of a field, the position an argument is read
	// from. What tells Imm from Ref is only this kind — both are runs of bytes, and 0x03 is
	// the number three under one and the value labelled "03" under the other.
	Imm

	// Name is a name that outlives the instruction writing it: something a scope bound, or
	// a scope to call. It is resolved by whoever runs the program rather than by position,
	// which is why it is not a Ref — a Ref points inside one stretch of instructions, and a
	// Name reaches across scopes and across modules.
	Name

	// Target is where control goes. Today it carries a count of instructions, which is what
	// the evaluator's cursor takes, and that count is why the instruction list cannot be
	// reordered: a pass that moves or inserts anything makes every count a lie. Naming it a
	// target is what lets it become the name of a block instead, which is the point at
	// which moving instructions becomes safe.
	Target

	// Text is bytes written for a person to read, and never a value. The message of an
	// assertion is the only one so far. It exists so that nothing tries to do arithmetic on
	// a sentence, and so that a tape width has no say over how long the sentence may be.
	Text
)

// An Operand is one of the two things an instruction points at, and what it is.
type Operand struct {
	kind  Kind
	bytes []byte
}

// Kind answers how these bytes are meant to be read.
func (o Operand) Kind() Kind { return o.kind }

// Bytes answers the operand as the bytes it carries.
//
// An operand that is not there answers with an empty slice rather than nil, because that is
// what every reader of the IR has always been handed and none of them checks.
func (o Operand) Bytes() []byte {
	if o.bytes == nil {
		return make([]byte, 0)
	}
	return o.bytes
}

// RefTo points at a value another instruction left behind, under its label.
func RefTo(label []byte) Operand { return Operand{Ref, label} }

// ImmOf carries the bytes themselves — a tape, or anything already in the shape it is used in.
func ImmOf(value []byte) Operand { return Operand{Imm, value} }

// ImmNum carries a number: a width, an index, a count.
func ImmNum(n uint64) Operand { return Operand{Imm, byteutil.FromUint64(n)} }

// NameOf carries a name, which outlives the instruction that writes it.
func NameOf(name string) Operand { return Operand{Name, []byte(name)} }

// TargetAt carries where control goes. It is still counted in instructions, which is what the
// evaluator's cursor takes; saying that it is a target is what will let it become something
// a pass can move without breaking.
func TargetAt(n uint64) Operand { return Operand{Target, byteutil.FromUint64(n)} }

// TextOf carries bytes written for a person: the message of an assertion, and nothing else
// so far.
func TextOf(text string) Operand { return Operand{Text, []byte(text)} }

// Nothing is the operand of an instruction that only points at one thing, or at none.
func Nothing() Operand { return Operand{Empty, nil} }

// String names a kind for a person reading the IR.
func (k Kind) String() string {
	switch k {
	case Ref:
		return "ref"
	case Imm:
		return "imm"
	case Name:
		return "name"
	case Target:
		return "target"
	case Text:
		return "text"
	}
	return "-"
}

// String writes an operand as what it is and what it carries, which is how a person reads one.
func (o Operand) String() string {
	if o.kind == Empty {
		return "-"
	}
	return o.kind.String() + " " + byteutil.ToHexPretty(o.bytes)
}
