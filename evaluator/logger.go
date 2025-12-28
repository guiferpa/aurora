package evaluator

import (
	"fmt"

	"github.com/guiferpa/aurora/emitter"
)

type Logger struct {
	enableLogging bool
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}

func (l *Logger) Println(opcode byte) {
	if l.enableLogging {
		fmt.Println(emitter.ResolveOpCode(opcode))
	}
}
