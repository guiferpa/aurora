package evm

import (
	"fmt"

	"github.com/guiferpa/aurora/wire/ir"
)

// What the builder writes, and what it has to say about the rest.
//
// A contract using something the builder does not write compiles, deploys and does nothing on
// chain — the worst way to find out. The list lives here, beside the code that does the
// writing: the emitter cannot hold it, because a phase knows nothing of the one after it, and
// the host cannot hold it either, because what a backend covers is not the host's to know.

// handled is every instruction the builder turns into opcodes or consumes as structure.
//
// OpDefer and OpBeginScope write nothing of their own and are not gaps: the first becomes an
// entry in the dispatcher, and the second opens a scope the builder lays out flat.
var handled = map[byte]bool{
	ir.OpAdd:        true,
	ir.OpSubtract:   true,
	ir.OpMultiply:   true,
	ir.OpDivide:     true,
	ir.OpSave:       true,
	ir.OpIdent:      true,
	ir.OpLoad:       true,
	ir.OpGetFeed:    true,
	ir.OpReturn:     true,
	ir.OpDefer:      true,
	ir.OpBeginScope: true,
}

// offChain is what is meant to be absent from a chain. Saying so is still worth a line: a
// program whose logs vanish should hear it from the compiler rather than from the silence.
var offChain = map[byte]string{
	ir.OpPrintBytes:   "printb writes a log, and a chain has nowhere to put one: it produces no bytecode, by decision",
	ir.OpPrintChars:   "printc writes a log, and a chain has nowhere to put one: it produces no bytecode, by decision",
	ir.OpPrintDecimal: "printd writes a log, and a chain has nowhere to put one: it produces no bytecode, by decision",
	ir.OpAssert:       "assert belongs to 'aurora test' and produces no bytecode, by decision",
}

// pending is what the builder does not write yet, named as the user wrote it. Instructions
// that come from one feature share a name, so an "if" is one warning rather than two.
var pending = map[byte]string{
	ir.OpIf:       "if",
	ir.OpJump:     "if",
	ir.OpCall:     "calling a scope",
	ir.OpPushFeed: "calling a scope",
	ir.OpPreCall:  "calling a scope",
	ir.OpDiff:     "a comparison",
	ir.OpEquals:   "a comparison",
	ir.OpBigger:   "a comparison",
	ir.OpSmaller:  "a comparison",
	ir.OpAnd:      "and/or",
	ir.OpOr:       "and/or",

	ir.OpExponential: "^",

	ir.OpPull: "a tape operation",
	ir.OpPush: "a tape operation",
	ir.OpHead: "a tape operation",
	ir.OpTail: "a tape operation",

	ir.OpJoin:  "struct",
	ir.OpField: "struct",
}

// A Warning is what the builder has to say about a program it cannot carry whole.
//
// It is the builder's own word rather than the emitter's. A backend reporting on itself in
// the vocabulary of the phase before it is one step from depending on that phase for meaning,
// and the whole point of the layering is that someone could take this package and write a
// different compiler around it.
//
// It carries no position: the IR has instructions, not lines.
type Warning struct {
	Message string
}

func (w Warning) String() string {
	return w.Message
}

// Warnings reports what a program uses that does not reach the bytecode.
//
// They are warnings and not errors because the backend is being written in slices: refusing
// the program would refuse it for a reason that expires. What must not happen is the binary
// coming out quietly wrong, which is what happened until now.
//
// They arrive in the order the program uses them, once each.
func Warnings(insts []ir.Instruction) []Warning {
	warnings := make([]Warning, 0)
	said := make(map[string]bool)

	for _, inst := range insts {
		op := inst.GetOpCode()
		if handled[op] {
			continue
		}

		message, ok := offChain[op]
		if !ok {
			name, pendingOp := pending[op]
			if !pendingOp {
				continue
			}
			message = fmt.Sprintf(
				"%s does not reach the bytecode yet: a contract using it compiles and does nothing on chain", name)
		}

		if said[message] {
			continue
		}
		said[message] = true
		warnings = append(warnings, Warning{Message: message})
	}

	return warnings
}
