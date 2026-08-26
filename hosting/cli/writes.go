package cli

import (
	"fmt"

	"github.com/guiferpa/aurora/wire/ir"
)

// Whether a scope of a program changes anything, asked before a chain is spoken to.
//
// It is what keeps a call and a transaction from being the same mistake twice. A call is a
// question — answered against the state as it is, costing nothing, and throwing away anything
// the answer did on the way. A scope that writes, asked as a question, is a write thrown away:
// the answer comes back, looking right, and the chain is exactly as it was.
//
// That happened, on a real chain, before this existed. `inc` answered 1 every time.

// WritesInput is what deciding takes: a program, and the name of the scope being reached.
type WritesInput struct {
	Blocks   []ir.Block
	Function string
}

// Writes says whether reaching this scope changes anything the chain keeps.
//
// It follows the scopes this one calls, because what a call does is part of what its caller
// does — and once there is a standard library, the scope somebody writes holds no `sstore` at
// all. `s.set(...)` is a call, and the write is one file away.
//
// A name the program does not bind answers false, and says so: the caller decides whether not
// finding it is worth stopping for, since a contract may have been deployed from a source that
// has since changed.
func ScopeWrites(in WritesInput) (writes bool, found bool) {
	scope, bound := ir.Scopes(in.Blocks)[in.Function]
	if !bound {
		return false, false
	}
	return ir.Does(in.Blocks, scope) >= ir.Writes, true
}

// refuseACallThatWrites is the message somebody gets for asking a question of something that
// changes an answer.
//
// It names the command that does what they meant, because being told what is wrong and not
// what to type is half an error message.
func refuseACallThatWrites(function string) error {
	return fmt.Errorf(
		"%s changes what the chain keeps, and a call is a question: it would answer, cost nothing, and leave the chain exactly as it was — send it with 'aurora tx %s' instead",
		function, function)
}

// warnATxThatWritesNothing is what somebody gets for paying for an answer they could have had.
//
// It is a word and not a refusal: a transaction over a scope that only reads is wasteful and
// not wrong, and there are reasons for one — a receipt, a block number, a timestamp that
// somebody else will read.
func warnATxThatWritesNothing(function string) string {
	return fmt.Sprintf(
		"%s changes nothing the chain keeps, so this costs gas for an answer 'aurora call %s' gives for free.",
		function, function)
}
