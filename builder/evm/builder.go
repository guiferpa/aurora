package evm

import (
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

const BYTE_SIZE = 8

type Builder struct {
	cursor       int
	insts        []emitter.Instruction
	operands     [][]byte
	logger       *Logger
	offsetIdents map[string]byte
}

func (t *Builder) writeSave(w io.Writer, left []byte) (int, error) {
	if len(left) == 1 {
		if _, err := t.writeBool(w, left[0]); err != nil {
			return 0, err
		}
	}
	t.operands = append(t.operands, left)
	return 0, nil
}

func (t *Builder) writeIdent(w io.Writer, ident []byte) (int, error) {
	if _, err := t.writePush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	offset := byte(len(t.offsetIdents) * 32)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	t.offsetIdents[string(ident)] = offset
	return 0, nil
}

func (t *Builder) writeLoad(w io.Writer, left []byte) (int, error) {
	offset := t.offsetIdents[string(left)]
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryLoad}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) writeGetArg(w io.Writer, left []byte) (int, error) {
	index := byteutil.ToUint64(left)
	offset := GetCalldataArgsOffset(index)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpCallDataLoad}); err != nil {
		return 0, err
	}
	return 0, nil
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

	bs := append(ic.Bytes(), rc.Bytes()...)
	if err := t.logger.Scanln(bs); err != nil {
		return 0, err
	}
	return w.Write(bs)
}

type NewBuilderOptions struct {
	EnableLogging bool
}

func NewBuilder(insts []emitter.Instruction, options NewBuilderOptions) *Builder {
	return &Builder{
		operands:     make([][]byte, 0),
		cursor:       0,
		insts:        insts,
		offsetIdents: make(map[string]byte),
		logger:       NewLogger(options.EnableLogging),
	}
}
