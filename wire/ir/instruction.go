package ir

type Instruction interface {
	GetLabel() []byte
	GetOpCode() byte
	GetLeft() Operand
	GetRight() Operand
}

type inst struct {
	label  []byte
	opcode byte
	left   Operand
	right  Operand
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

// NewInstruction builds an instruction from operands that say what they are.
func NewInstruction(label []byte, opcode byte, left, right Operand) inst {
	return inst{label, opcode, left, right}
}
