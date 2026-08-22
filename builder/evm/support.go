package evm

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/diag"
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
//
// OpIdent is here for the one it does carry — the binding that names a deferred scope, which
// the dispatcher reads as a selector. The one inside a scope it does not, and that is what
// bindingsInsideScopes answers for: a per-opcode list cannot tell the two apart, because
// which one it is depends on where the instruction sits.
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
	ir.OpIf:      "if",
	ir.OpJump:    "if",
	ir.OpCall:    "calling a scope",
	ir.OpPreCall: "calling a scope",
	ir.OpDiff:    "a comparison",
	ir.OpEquals:  "a comparison",
	ir.OpBigger:  "a comparison",
	ir.OpSmaller: "a comparison",
	ir.OpAnd:     "and/or",
	ir.OpOr:      "and/or",

	ir.OpExponential: "^",

	ir.OpPull: "a tape operation",
	ir.OpPush: "a tape operation",
	ir.OpHead: "a tape operation",
	ir.OpTail: "a tape operation",

	ir.OpJoin:  "shape",
	ir.OpField: "shape",
}

// bindingInsideAScope reports the first name bound inside a deferred scope, if there is one.
//
// It is its own walk because the list of handled opcodes cannot answer it. An OpIdent at the
// top of a program names a deferred scope and becomes a selector the dispatcher reads; the
// same opcode inside a scope stores a value, and the lowering hands that value to nothing, so
// the MSTORE writing it finds an empty stack. Which one an instruction is depends on where it
// sits, and a map keyed by opcode has no way to see that.
//
// Until now the map said OpIdent was handled and nothing was said at all: a scope binding a
// name compiled, deployed, and answered a different number than the same program answers off
// the chain. That is the worst outcome this file exists to prevent, and it was the one case
// getting past it.
func bindingInsideAScope(insts []ir.Instruction) (diag.Warning, bool) {
	// A defer says how many instructions its body is, so its range is what has to be looked
	// inside. Bodies nest, and the deepest end is what closes the outermost — a body that
	// holds a defer holds that defer's body too.
	end := 0
	for at, inst := range insts {
		switch inst.GetOpCode() {
		case ir.OpDefer:
			body := at + 1 + int(byteutil.ToUint64(inst.GetRight().Bytes()))
			if body > end {
				end = body
			}
		case ir.OpIdent:
			if at >= end {
				continue
			}
			warning := diag.Warning{Message: "a name bound inside a scope does not reach the bytecode: " +
				"the value it stores is dropped, and the contract answers something else than the program does"}
			if origin := inst.GetOrigin(); origin.Known() {
				warning.Line, warning.Column = origin.Line, origin.Column
			}
			return warning, true
		}
	}
	return diag.Warning{}, false
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
func Warnings(insts []ir.Instruction) []diag.Warning {
	warnings := make([]diag.Warning, 0)
	said := make(map[string]bool)

	if warning, found := bindingInsideAScope(insts); found {
		warnings = append(warnings, warning)
	}

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
		warning := diag.Warning{Message: message}
		if origin := inst.GetOrigin(); origin.Known() {
			warning.Line, warning.Column = origin.Line, origin.Column
		}
		warnings = append(warnings, warning)
	}

	return warnings
}
