package emitter

import "fmt"

type Logger struct {
	enableLogging bool
}

func (l *Logger) Println(insts []Instruction) {
	if l.enableLogging {
		for _, inst := range insts {
			fmt.Println(ResolveOpCode(inst.GetOpCode()))
		}
	}
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}
