package ir

import (
	"bytes"
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// Format renders instructions one per line, which is how the IR is read by a person.
//
// It lives with the IR rather than with whoever shows it: writing the vocabulary down is part
// of the vocabulary, the way a token knows how to spell itself. What belongs to a host is the
// decision to show it and the writer it goes to.
//
// It returns the text rather than printing it: a package that writes takes that decision away
// — and it also gets in the way of a test, which wants the string to compare.
func Format(insts []Instruction) string {
	bs := bytes.NewBuffer(make([]byte, 0))
	for _, inst := range insts {
		fmt.Fprintf(bs, "%s %s %s %s\n", byteutil.ToHexPretty(inst.GetLabel()), ResolveOpCode(inst.GetOpCode()), inst.GetLeft(), inst.GetRight())
	}
	return bs.String()
}
