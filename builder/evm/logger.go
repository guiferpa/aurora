package evm

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
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

func (l *Logger) printReturn(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("RETURN"), "")
}

func (l *Logger) printMemoryStore(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("MSTORE"), "")
}

func (l *Logger) printStop(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("STOP"), "")
}

func (l *Logger) printCodeCopy(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("CODECOPY"), "")
}

func (l *Logger) printCallDataLoad(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("CALLDATALOAD"), "")
}

func (l *Logger) printKECCAK256(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("KECCAK256"), "")
}

func (l *Logger) printShiftRight(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("SHR"), "")
}

func (l *Logger) printEqual(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("EQ"), "")
}

func (l *Logger) printJumpIf(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("JUMPI"), "")
}

func (l *Logger) printJumpDestiny(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("JUMPDEST"), "")
}

func (l *Logger) printAdd(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("ADD"), "")
}

func (l *Logger) printMul(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("MUL"), "")
}

func (l *Logger) printSub(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("SUB"), "")
}

func (l *Logger) printDiv(w *tabwriter.Writer) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("DIV"), "")
}

func (l *Logger) printPush1(w *tabwriter.Writer, param []byte) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("PUSH1"), magenta(byteutil.ToHexBloom(param)))
}

func (l *Logger) printPush4(w *tabwriter.Writer, param []byte) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("PUSH4"), magenta(byteutil.ToHexBloom(param)))
}

func (l *Logger) printPush8(w *tabwriter.Writer, param []byte) (int, error) {
	return fmt.Fprintf(w, "%s\t%s\n", yellow("PUSH8"), magenta(byteutil.ToHexBloom(param)))
}

func (l *Logger) Scanln(bs []byte) error {
	if l.enableLogging {
		i := 0
		for len(bs) > i {
			opcode := bs[i]
			if opcode == OpPush1 {
				if _, err := l.printPush1(l.w, []byte{bs[i+1]}); err != nil {
					return err
				}
				i += 2
				continue
			}
			if opcode == OpPush4 {
				if _, err := l.printPush4(l.w, bs[i+1:i+5]); err != nil {
					return err
				}
				i += 5
				continue
			}
			if opcode == OpPush8 {
				if _, err := l.printPush8(l.w, bs[i+1:i+9]); err != nil {
					return err
				}
				i += 9
				continue
			}
			if opcode == OpCodeCopy {
				if _, err := l.printCodeCopy(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpReturn {
				if _, err := l.printReturn(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpMemoryStore {
				if _, err := l.printMemoryStore(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpStop {
				if _, err := l.printStop(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpKECCAK256 {
				if _, err := l.printKECCAK256(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpCallDataLoad {
				if _, err := l.printCallDataLoad(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpShiftRight {
				if _, err := l.printShiftRight(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpEqual {
				if _, err := l.printEqual(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpAdd {
				if _, err := l.printAdd(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpMul {
				if _, err := l.printMul(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpSub {
				if _, err := l.printSub(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpDiv {
				if _, err := l.printDiv(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpJumpIf {
				if _, err := l.printJumpIf(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			if opcode == OpJumpDestiny {
				if _, err := l.printJumpDestiny(l.w); err != nil {
					return err
				}
				i += 1
				continue
			}
			fmt.Println("opcode", ResolveOpCode(opcode), "param", byteutil.ToHexBloom(bs[i:]))
		}
		return l.w.Flush()
	}
	return nil
}

func (l *Logger) Close() error {
	return l.w.Flush()
}
