package print

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

func Opcodes(w io.Writer, ocs []emitter.OpCode) error {
	for _, oc := range ocs {
		bs := fmt.Sprintf("%v %v %v %v", oc.Label, oc.Operation, oc.Left, oc.Right)
		fmt.Fprintf(w, "%s: %s %s %s --> %v\n", oc.Label, oc.Operation, oc.Left, oc.Right, bs)
	}
	return nil
}
