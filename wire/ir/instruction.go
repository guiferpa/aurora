package ir

type Instruction interface {
	GetLabel() []byte
	GetOpCode() byte
	GetLeft() Operand
	GetRight() Operand
	GetOperands() []Operand
	GetOrigin() Origin
}

type inst struct {
	label    []byte
	opcode   byte
	operands []Operand
	origin   Origin
}

func (i inst) GetLabel() []byte {
	return i.label
}

func (i inst) GetOpCode() byte {
	return i.opcode
}

// GetLeft and GetRight answer the first two operands, which is what an instruction that takes
// a fixed pair has and all an older reader ever asked for. An operand that is not there
// answers as Nothing.
func (i inst) GetLeft() Operand {
	return i.operandAt(0)
}

func (i inst) GetRight() Operand {
	return i.operandAt(1)
}

// GetOperands answers every operand, in the order they were written. A construction of n
// values — the fields of a shape, the items of a tape — is one instruction with n operands
// rather than a chain of n instructions, and this is how it is read.
func (i inst) GetOperands() []Operand {
	return i.operands
}

func (i inst) operandAt(at int) Operand {
	if at >= len(i.operands) {
		return Nothing()
	}
	return i.operands[at]
}

func (i inst) GetOrigin() Origin {
	return i.origin
}

// At answers the same instruction, knowing where it came from.
//
// It is a second step rather than a parameter because most instructions have somewhere to
// point at and a few do not, and because an origin changes nothing about what the instruction
// does — reading "NewInstruction(...).At(...)" says that, and a fifth argument would not.
func (i inst) At(origin Origin) inst {
	i.origin = origin
	return i
}

// NewInstruction builds an instruction that takes a pair of operands, which is most of them.
func NewInstruction(label []byte, opcode byte, left, right Operand) inst {
	return inst{label: label, opcode: opcode, operands: []Operand{left, right}}
}

// NewInstructionOver builds an instruction that takes as many operands as it was given.
//
// A construction is the case for it: a shape has as many fields as it has, and a tape literal
// as many items. Both used to be a chain of two-operand instructions, which left whoever read
// the IR to recognise that the chain was one thing — and recognising shapes is what a
// consumer of an IR should never have to do.
func NewInstructionOver(label []byte, opcode byte, operands ...Operand) inst {
	return inst{label: label, opcode: opcode, operands: operands}
}
