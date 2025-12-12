package evm

import (
	"bytes"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type RuntimeCodeReferenced struct {
	Label        string
	Instructions []emitter.Instruction
}

type RuntimeCode struct {
	Fallback   []emitter.Instruction
	Referenced []RuntimeCodeReferenced
}

func (t *Builder) pickReferencedCode() []emitter.Instruction {
	instructions := make([]emitter.Instruction, 0)
	for t.cursor < len(t.insts) {
		inst := t.insts[t.cursor]
		if inst.GetOpCode() == emitter.OpReturn {
			return instructions
		}
		instructions = append(instructions, inst)
		t.cursor++
	}
	return instructions
}

func (t *Builder) pickRuntimeCode() RuntimeCode {
	referenced := make([]RuntimeCodeReferenced, 0)
	fallback := make([]emitter.Instruction, 0)
	for t.cursor < len(t.insts) {
		inst := t.insts[t.cursor]
		if inst.GetOpCode() == emitter.OpIdent {
			label := string(inst.GetLeft())
			instructions := t.pickReferencedCode()
			referenced = append(referenced, RuntimeCodeReferenced{Label: label, Instructions: instructions})
		} else {
			fallback = append(fallback, inst)
		}
		t.cursor++
	}
	return RuntimeCode{Referenced: referenced, Fallback: fallback}
}

func (t *Builder) writeReferencedCode(bs io.Writer, insts []emitter.Instruction) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

		/*
			if op == emitter.OpIdent {
				id := string(inst.GetLeft())
				if _, err := t.buildIdent(bs, fmt.Sprintf("%s()", id)); err != nil {
					return nil, err
				}
			}
		*/

		if op == emitter.OpAdd {
			return t.buildAdd(bs)
		}

		if op == emitter.OpMultiply {
			return t.buildMult(bs)
		}

		if op == emitter.OpSubtract {
			return t.buildSub(bs)
		}

		if op == emitter.OpDivide {
			return t.buildDiv(bs)
		}

		if op == emitter.OpSave {
			t.operands = append(t.operands, inst.GetLeft())
		}
	}

	return 0, nil
}

func (t *Builder) writeRuntimeCode() (*bytes.Buffer, error) {
	rc := t.pickRuntimeCode()
	referenced := rc.Referenced

	bs := bytes.NewBuffer(make([]byte, 0))

	for _, referenced := range referenced {
		writeDispatcher(bs, referenced.Label)
	}
	writeDispatcher(bs, "root")

	for _, referenced := range referenced {
		if _, err := t.writeReferencedCode(bs, referenced.Instructions); err != nil {
			return nil, err
		}
	}
	t.writeReferencedCode(bs, rc.Fallback)

	return bs, nil
}
