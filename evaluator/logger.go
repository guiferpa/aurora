package evaluator

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

var yellow = color.New(color.FgHiYellow).SprintFunc()
var magenta = color.New(color.FgHiMagenta).SprintFunc()

type Logger struct {
	enableLogging bool
	w             *tabwriter.Writer
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{
		enableLogging: enableLogging,
		w:             tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

func (l *Logger) Println(inst emitter.Instruction) error {
	label := inst.GetLabel()
	opcode := inst.GetOpCode()
	left := inst.GetLeft()
	right := inst.GetRight()
	if l.enableLogging {
		resolveOpcode := color.New(color.FgHiCyan).Sprint(emitter.ResolveOpCode(opcode))
		colorizedLeft := magenta("<empty>")
		if len(left) > 0 {
			colorizedLeft = yellow(byteutil.ToHexBloom(left))
		}
		colorizedRight := magenta("<empty>")
		if len(right) > 0 {
			colorizedRight = yellow(byteutil.ToHexBloom(right))
		}
		labelHex := yellow(byteutil.ToHexBloom(label))
		if _, err := fmt.Fprintf(l.w, "%s\t%s\t%s\t%s\n", labelHex, resolveOpcode, colorizedLeft, colorizedRight); err != nil {
			return err
		}
	}
	return nil
}

func (l *Logger) Close() error {
	return l.w.Flush()
}
