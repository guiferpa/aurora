package evm

import (
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

func ToOpByte(op uint32) []byte {
	return byteutil.NoPadding(byteutil.FromUint32(op))
}

type Builder struct {
	operands [][]byte
}

func (t *Builder) push(value []byte) {
	t.operands = append(t.operands, value)
}

func (t *Builder) pop() []byte {
	value := t.operands[len(t.operands)-1]
	t.operands = t.operands[:len(t.operands)-1]
	return value
}

func (t *Builder) emitPush8Safe(w io.Writer) (int, error) {
	if len(t.operands) == 0 {
		return 0, nil
	}
	if _, err := w.Write([]byte{OpPush8}); err != nil {
		return 0, err
	}
	operand := t.pop()
	if _, err := w.Write(operand); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) emitAddOperation(w io.Writer) (int, error) {
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpAdd}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) emitMultiplyOperation(w io.Writer) (int, error) {
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMul}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) emitSubtractOperation(w io.Writer) (int, error) {
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpSub}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) emitDivideOperation(w io.Writer) (int, error) {
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := t.emitPush8Safe(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpDiv}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) Build(w io.Writer, insts []emitter.Instruction) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

		if op == emitter.OpAdd {
			if _, err := t.emitAddOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpMultiply {
			if _, err := t.emitMultiplyOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpSubtract {
			if _, err := t.emitSubtractOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpDivide {
			if _, err := t.emitDivideOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpSave {
			t.push(inst.GetLeft())
		}
	}
	return 0, nil
}

func NewBuilder() *Builder {
	return &Builder{
		operands: make([][]byte, 0),
	}
}
