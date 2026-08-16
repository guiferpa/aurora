package emitter

import (
	"bytes"
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// Format renders instructions one per line, which is how the IR is read by a person.
//
// It returns the text rather than printing it: showing something is the host's decision,
// and a phase that writes takes that decision away — it also gets in the way of a test,
// which wants the string to compare.
func Format(insts []Instruction) string {
	bs := bytes.NewBuffer(make([]byte, 0))
	for _, inst := range insts {
		fmt.Fprintf(bs, "%s %s %s %s\n", byteutil.ToHexPretty(inst.GetLabel()), ResolveOpCode(inst.GetOpCode()), byteutil.ToHexPretty(inst.GetLeft()), byteutil.ToHexPretty(inst.GetRight()))
	}
	return bs.String()
}
