package print

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

func Instructions(w io.Writer, insts []emitter.Instruction, is bool) error {
	for _, ins := range insts {
		fmt.Fprintf(w, "%x(%d): %x(1) %x(%d) %x(%d)\n", ins.GetLabel(), len(ins.GetLabel()), ins.GetOpCode(), ins.GetLeft(), len(ins.GetLeft()), ins.GetRight(), len(ins.GetRight()))
	}
	return nil
}
