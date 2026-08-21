package ir

type Instruction interface {
	GetLabel() []byte
	GetOpCode() byte
	GetLeft() Operand
	GetRight() Operand
	GetOrigin() Origin
}

type inst struct {
	label  []byte
	opcode byte
	left   Operand
	right  Operand
	origin Origin
}

func (i inst) GetLabel() []byte {
	return i.label
}

func (i inst) GetOpCode() byte {
	return i.opcode
}

func (i inst) GetLeft() Operand {
	return i.left
}

func (i inst) GetRight() Operand {
	return i.right
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

// NewInstruction builds an instruction from operands that say what they are.
func NewInstruction(label []byte, opcode byte, left, right Operand) inst {
	return inst{label: label, opcode: opcode, left: left, right: right}
}
