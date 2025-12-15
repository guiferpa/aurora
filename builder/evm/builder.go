package evm

import (
	"io"

	"github.com/guiferpa/aurora/emitter"
)

const CALLDATA_SLOT_READABLE = 32 // bytes
const BYTE_SIZE = 8

func GetCalldataArgsIndex(index int) byte {
	return CALLDATA_SLOT_READABLE << index
}

type Builder struct {
	cursor   int
	insts    []emitter.Instruction
	operands [][]byte
}

func (t *Builder) writePush8SafeFromOperands(w io.Writer) (int, error) {
	if len(t.operands) == 0 {
		return 0, nil
	}
	if _, err := w.Write([]byte{OpPush8}); err != nil {
		return 0, err
	}
	operand := t.operands[len(t.operands)-1]
	t.operands = t.operands[:len(t.operands)-1]
	if _, err := w.Write(operand); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) Build(w io.Writer) (int, error) {
	rc, err := t.buildRuntimeCode()
	if err != nil {
		return 0, err
	}

	ic, err := t.buildInstantiateCode(byte(rc.Len()))
	if err != nil {
		return 0, err
	}

	return w.Write(append(ic.Bytes(), rc.Bytes()...))
}

func NewBuilder(insts []emitter.Instruction) *Builder {
	return &Builder{
		operands: make([][]byte, 0),
		cursor:   0,
		insts:    insts,
	}
}
