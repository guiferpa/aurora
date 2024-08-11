package emitter

type Operation string

const (
	OpSave Operation = "SAV"
	OpSum  Operation = "SUM"
	OpSub  Operation = "SUB"
	OpMult Operation = "MUL"
	OpDiv  Operation = "DIV"
)

type OpCode struct {
	Label     string
	Operation Operation
	Left      []byte
	Right     []byte
}
