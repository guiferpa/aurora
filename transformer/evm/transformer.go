package evm

import (
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Transformer struct{}

func (t *Transformer) Transform(w io.Writer, insts []emitter.Instruction) error {
	return nil
}

func NewTransformer() *Transformer {
	return &Transformer{}
}
