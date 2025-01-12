package emitter

import (
	"encoding/binary"
)

const (
	LINE_SIZE_COUNTER  = 4
	LABEL_SIZE_COUNTER = 4
	OP_SIZE_COUNTER    = 1
	LEFT_SIZE_COUNTER  = 4
	RIGHT_SIZE_COUNTER = 4
)

func parseLine(ln []byte) (Instruction, error) {
	// Extract label
	lblen := binary.BigEndian.Uint32((ln[0:LABEL_SIZE_COUNTER]))
	label := ln[LABEL_SIZE_COUNTER : lblen+LABEL_SIZE_COUNTER]
	ln = ln[lblen+LABEL_SIZE_COUNTER:]

	// Extract opcode
	op := ln[0:OP_SIZE_COUNTER]
	ln = ln[OP_SIZE_COUNTER:]

	// Extract left
	lflen := binary.BigEndian.Uint32((ln[0:LEFT_SIZE_COUNTER]))
	left := ln[LEFT_SIZE_COUNTER : lflen+LEFT_SIZE_COUNTER]
	ln = ln[lflen+LEFT_SIZE_COUNTER:]

	// Extract right
	rglen := binary.BigEndian.Uint32((ln[0:RIGHT_SIZE_COUNTER]))
	right := ln[RIGHT_SIZE_COUNTER : rglen+RIGHT_SIZE_COUNTER]
	ln = ln[rglen+RIGHT_SIZE_COUNTER:]

	return NewInstruction(label, op[0], left, right), nil
}

func Parse(bs []byte) ([]Instruction, error) {
	insts := make([]Instruction, 0)

	for len(bs) > 0 {
		lnlen := binary.BigEndian.Uint32((bs[0:LINE_SIZE_COUNTER]))
		line := bs[LINE_SIZE_COUNTER : lnlen+LINE_SIZE_COUNTER]
		inst, err := parseLine(line)
		if err != nil {
			return insts, err
		}
		insts = append(insts, inst)
		bs = bs[lnlen+LINE_SIZE_COUNTER:]
	}

	return insts, nil
}
