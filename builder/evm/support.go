package evm

import (
	"fmt"

	"github.com/guiferpa/aurora/wire/diag"
	"github.com/guiferpa/aurora/wire/ir"
)

// What the builder writes, and what it has to say about the rest.
//
// A contract using something the builder does not write compiles, deploys and does nothing on
// chain — the worst way to find out. The list lives here, beside the code that does the
// writing: the emitter cannot hold it, because a phase knows nothing of the one after it, and
// the host cannot hold it either, because what a backend covers is not the host's to know.

// handled is every instruction the builder turns into opcodes.
//
// Nothing here is structure. Where a program goes is what a block's terminator says, and an
// instruction computes a value and does nothing else — so the list used to hold five opcodes
// that wrote no bytes and were not gaps, and now it holds none.
var handled = map[byte]bool{
	ir.OpAdd:         true,
	ir.OpSubtract:    true,
	ir.OpMultiply:    true,
	ir.OpDivide:      true,
	ir.OpSave:        true,
	ir.OpIdent:       true,
	ir.OpLoad:        true,
	ir.OpGetFeed:     true,
	ir.OpEquals:      true,
	ir.OpDiff:        true,
	ir.OpBigger:      true,
	ir.OpSmaller:     true,
	ir.OpAnd:         true,
	ir.OpOr:          true,
	ir.OpExponential: true,
	ir.OpCall:        true,
	ir.OpJoin:        true,
	ir.OpField:       true,
	ir.OpPull:        true,
	ir.OpPush:        true,
	ir.OpHead:        true,
	ir.OpTail:        true,
}

// offChain is what is meant to be absent from a chain. Saying so is still worth a line: a
// program whose logs vanish should hear it from the compiler rather than from the silence.
//
// A print is ignored rather than refused, and what is left of it is the value it was given. It
// is an expression like any other and is worth what it showed, so a print written into a
// working program to see what a value is does not change what that program answers — on a
// chain it is the identity, and off one it is the identity that also says something.
var offChain = map[byte]string{
	ir.OpPrintBytes:   "printb is ignored in compiled code, by decision: a chain has nowhere to put a log, and the value it was given carries on",
	ir.OpPrintChars:   "printc is ignored in compiled code, by decision: a chain has nowhere to put a log, and the value it was given carries on",
	ir.OpPrintDecimal: "printd is ignored in compiled code, by decision: a chain has nowhere to put a log, and the value it was given carries on",
	ir.OpAssert:       "assert belongs to 'aurora test' and produces no bytecode, by decision",
}

// pending is what the builder does not write yet, named as the user wrote it rather than as
// the IR names it. Instructions that come from one feature share a name, so a feature is one
// warning rather than one per opcode it lowers to.
//
// An opcode in neither list is warned about under the name the IR gives it rather than passed
// over: a gap nobody remembered to write down here is still a gap, and the point of this file
// is that a gap is never silent.
var pending = map[byte]string{
	ir.OpStorageGet: "sload does not reach the bytecode yet: a contract using it compiles, and reads the neutral tape on chain",
	ir.OpStorageSet: "sstore does not reach the bytecode yet: a contract using it compiles, and keeps nothing on chain",
}

// Warnings reports what a program uses that does not reach the bytecode.
//
// They are warnings and not errors because the backend is being written in slices: refusing
// the program would refuse it for a reason that expires. What must not happen is the binary
// coming out quietly wrong, which is what happened until now.
//
// They arrive in the order the program uses them, once each, and each one points at the first
// place the program used it. That place comes from the instruction: the emitter knows where
// every node was written and now says so, where before this named a feature and left the
// person to find it.
func Warnings(blocks []ir.Block) []diag.Warning {
	insts := make([]ir.Instruction, 0)
	for _, block := range blocks {
		insts = append(insts, block.Insts...)
	}

	warnings := make([]diag.Warning, 0)
	said := make(map[string]bool)

	for _, inst := range insts {
		op := inst.GetOpCode()
		if handled[op] {
			continue
		}

		message, ok := offChain[op]
		if !ok {
			name, named := pending[op]
			if !named {
				name = ir.ResolveOpCode(op)
			}
			message = fmt.Sprintf(
				"%s does not reach the bytecode yet: a contract using it compiles and does nothing on chain", name)
		}

		if said[message] {
			continue
		}
		said[message] = true
		warning := diag.Warning{Message: message}
		if origin := inst.GetOrigin(); origin.Known() {
			warning.Line, warning.Column = origin.Line, origin.Column
		}
		warnings = append(warnings, warning)
	}

	return warnings
}
