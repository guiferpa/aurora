package transformer

import (
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Transformer interface {
	Transform(w io.Writer, insts []emitter.Instruction) error
}
