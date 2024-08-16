package print

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

func Opcodes(w io.Writer, ocs []emitter.OpCode, is bool) error {
	for _, oc := range ocs {
		if !is {
			fmt.Fprintf(w, "%x(%d): %x(%d) %x(%d) %x(%d)\n", oc.Label, len(oc.Label), oc.Operation, len(oc.Operation), oc.Left, len(oc.Left), oc.Right, len(oc.Right))
			continue
		}
			fmt.Fprintf(w, "%s(%d): %s(%d) %s(%d) %s(%d)\n", oc.Label, len(oc.Label), oc.Operation, len(oc.Operation), oc.Left, len(oc.Left), oc.Right, len(oc.Right))
	}
	return nil
}
