package lexer

import (
	"fmt"

	"os"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
)

type Logger struct {
	enableLogging bool
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}

func (l *Logger) Println(tag Tag, match []byte) (int, error) {
	if l.enableLogging {
		id := color.New(color.FgHiCyan).Sprint(tag.Id)
		bs := color.New(color.FgHiYellow).Sprint(byteutil.ToHexBloom(match))
		return fmt.Fprintf(os.Stdout, "%s: %s\n", id, bs)
	}
	return 0, nil
}
