package print

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

func Opcodes(w io.Writer, ocs []emitter.OpCode) error {
	for _, oc := range ocs {
		fmt.Fprintf(w, "%x(%d): %x(%d) %x(%d) %x(%d)\n", oc.Label, len(oc.Label), oc.Operation, len(oc.Operation), oc.Left, len(oc.Left), oc.Right, len(oc.Right))
	}
	return nil
}
