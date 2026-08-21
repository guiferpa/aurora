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
	Empty  Kind = iota // there is nothing here
	Ref                // the label of a value another instruction left behind
	Imm                // the bytes themselves: a tape, a width, an index
	Name               // a name bound in a scope, or a scope to call
	Target             // where control goes
	Text               // bytes written for a person to read
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
