package emitter

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

func resolveOpCode(op byte) string {
	switch op {
	case OpMultiply:
		return "OpMultiply"
	case OpAdd:
		return "OpAdd"
	case OpSubstract:
		return "OpSubstract"
	case OpDivide:
		return "OpDivide"
	case OpExponential:
		return "OpExponential"
	case OpIdentify:
		return "OpIdentify"
	case OpLoad:
		return "OpLoad"
	case OpBigger:
		return "OpBigger"
	case OpDiff:
		return "OpDiff"
	case OpEquals:
		return "OpEquals"
	case OpSmaller:
		return "OpSmaller"
	case OpBeginScope:
		return "OpBeginScope"
	case OpEndScope:
		return "OpEndScope"
	case OpSave:
		return "OpSave"
	case OpPreCall:
		return "OpPreCall"
	case OpCall:
		return "OpCall"
	case OpSaveParam:
		return "OpSaveParam"
	case OpPrint:
		return "OpPrint"
	case OpBeginFunc:
		return "OpBeginFunc"
	case OpEndFunc:
		return "OpEndFunc"
	case OpSetLocal:
		return "OpSetLocal"
	case OpGetLocal:
		return "OpGetLocal"
	case OpReturn:
		return "OpReturn"
	case OpResult:
		return "OpResult"
	case OpLoadParam:
		return "OpLoadParam"
	case OpIfNot:
		return "OpIfNot"
	case OpOr:
		return "OpOr"
	case OpAnd:
		return "OpAnd"
	case OpJump:
		return "OpJump"
	case OpPushArg:
		return "OpPushArg"
	case OpGetArg:
		return "OpGetArg"
	}
	return "%Unknown%"
}

func highlightBytesUsed(c *color.Color, param []byte, b int, f string) string {
	length := len(param)
	padding := b - length
	return fmt.Sprintf("%s%s", strings.Repeat("0", padding*2), c.Sprintf(f, param))
}

func highlightBytesUsedInHex(c *color.Color, param []byte, b int) string {
	return highlightBytesUsed(c, param, b, "%x")
}

func highlightByteUsedInHex(c *color.Color, param byte, b int) string {
	bs := make([]byte, 0)
	bs = append(bs, param)
	return highlightBytesUsedInHex(c, bs, b)
}

func Print(w io.Writer, insts []Instruction) error {
	c := color.New(color.FgHiYellow)
	for i, ins := range insts {
		lo := fmt.Sprintf("%-12s", resolveOpCode(ins.GetOpCode()))
		t := highlightBytesUsedInHex(c, ins.GetLabel(), 4)
		lp := highlightBytesUsedInHex(c, ins.GetLeft(), 8)
		rp := highlightBytesUsedInHex(c, ins.GetRight(), 8)
		o := highlightByteUsedInHex(c, ins.GetOpCode(), 1)
		fmt.Fprintf(w, "[%016s] %0s(%d): %s(1) %s - %s(%d) %s(%d)\n", c.Sprintf("%x", i), t, len(ins.GetLabel()), o, lo, lp, len(ins.GetLeft()), rp, len(ins.GetRight()))
	}
	return nil
}
