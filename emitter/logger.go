package emitter

import "fmt"

type Logger struct {
	enableLogging bool
}

func (l *Logger) Println(insts []Instruction) {
	if l.enableLogging {
		fmt.Println(insts)
	}
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}
