package emitter

type Instruction interface {
	GetLabel() []byte
	GetOpCode() byte
	GetLeft() []byte
	GetRight() []byte
}

type inst struct {
	label  []byte
	opcode byte
	left   []byte
	right  []byte
}

func (i inst) GetLabel() []byte {
	return i.label
}

func (i inst) GetOpCode() byte {
	return i.opcode
}

func (i inst) GetLeft() []byte {
	return i.left
}

func (i inst) GetRight() []byte {
	return i.right
}

func NewInstruction(label []byte, opcode byte, left, right []byte) inst {
	if left == nil {
		left = make([]byte, 0)
	}
	if right == nil {
		right = make([]byte, 0)
	}
	return inst{label, opcode, left, right}
}
