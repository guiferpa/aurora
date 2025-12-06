package builder

import (
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Builder interface {
	Builder(w io.Writer, insts []emitter.Instruction) error
}
