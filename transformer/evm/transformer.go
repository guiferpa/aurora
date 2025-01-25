package evm

import (
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

type Transformer struct{}

func (t *Transformer) Transform(w io.Writer, insts []emitter.Instruction) error {
	for _, inst := range insts {
		op := inst.GetOpCode()

		if op == emitter.OpAdd {
			if _, err := w.Write(byteutil.FromUint32(OpAdd)); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewTransformer() *Transformer {
	return &Transformer{}
}
