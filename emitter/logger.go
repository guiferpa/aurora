package emitter

import (
	"bytes"
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

type Logger struct {
	enableLogging bool
}

func Format(insts []Instruction) string {
	bs := bytes.NewBuffer(make([]byte, 0))
	for _, inst := range insts {
		fmt.Fprintf(bs, "%s %s %s %s\n", byteutil.ToHexPretty(inst.GetLabel()), ResolveOpCode(inst.GetOpCode()), byteutil.ToHexPretty(inst.GetLeft()), byteutil.ToHexPretty(inst.GetRight()))
	}
	return bs.String()
}

func (l *Logger) Println(insts []Instruction) {
	if l.enableLogging {
		fmt.Println(Format(insts))
	}
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}
