package print

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

func Instructions(w io.Writer, insts []emitter.Instruction, is bool) error {
	for i, ins := range insts {
		fmt.Fprintf(w, "[%d] %x(%d): %x(1) %x(%d) %x(%d)\n", i, ins.GetLabel(), len(ins.GetLabel()), ins.GetOpCode(), ins.GetLeft(), len(ins.GetLeft()), ins.GetRight(), len(ins.GetRight()))
	}
	return nil
}
