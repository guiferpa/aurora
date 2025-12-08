package evm

import (
	"io"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

const CALLDATA_SLOT_READABLE = 32 // bytes

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

func (t *Builder) buildPush8SafeFromOperands(w io.Writer) (int, error) {
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

func (t *Builder) buildAddOperation(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpAdd}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildMultiplyOperation(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMul}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildSubtractOperation(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpSub}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildDivideOperation(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpDiv}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildDispatcherOperation(w io.Writer, id string) (int, error) {
	if _, err := w.Write([]byte{OpPush1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{0x00}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpCallDataLoad}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{byte(CALLDATA_SLOT_READABLE - 4)}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpShiftRight}); err != nil {
		return 0, err
	}
	selector := crypto.Keccak256([]byte(id))[:4]
	if _, err := w.Write(selector); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) Build(w io.Writer, insts []emitter.Instruction) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

		if op == emitter.OpPreCall {
			if _, err := t.buildDispatcherOperation(w, "sum()"); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpAdd {
			if _, err := t.buildAddOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpMultiply {
			if _, err := t.buildMultiplyOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpSubtract {
			if _, err := t.buildSubtractOperation(w); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpDivide {
			if _, err := t.buildDivideOperation(w); err != nil {
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
