package evaluator

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// What a chain keeps between one transaction and the next, off a chain.
//
// The point of the backend is that the same program answers the same thing on a chain and off
// it, and storage is where that stops being about arithmetic: a program that reads what an
// earlier write left can only be simulated by something that keeps it. A map does, for as long
// as the evaluator lives, which is one run of a program.
//
// A second run starts empty and a chain does not, and that difference is not hidden: what is
// simulated here is what one transaction sees, which is exactly what simulating a call off a
// chain is for.

// EvaluateStorageGet answers what is kept under a key.
//
// A key nothing was ever written under answers the neutral tape, which is what a slot never
// written answers on a chain, and what reading past the end of a run answers here.
func (e *Evaluator) EvaluateStorageGet(label []byte, left, right ir.Operand) error {
	kept, written := e.storage[byteutil.ToHex(e.slot(left))]
	if !written {
		kept = byteutil.FalseTape(e.tapeSize)
	}
	e.environ.SetTemp(byteutil.ToHex(label), kept)
	return nil
}

// EvaluateStorageSet keeps a value under a key, and leaves the value.
//
// It leaves it because everything in Aurora is an expression and a write is no exception, so
// `sstore 1 41 + 1` is forty-two — and because a write that answered nothing would be the only
// instruction in the language that answers nothing.
func (e *Evaluator) EvaluateStorageSet(label []byte, left, right ir.Operand) error {
	value := byteutil.PaddingTape(e.value(right), e.tapeSize)
	e.storage[byteutil.ToHex(e.slot(left))] = value
	e.environ.SetTemp(byteutil.ToHex(label), value)
	return nil
}

// slot answers the tape a key is, as wide as every other value.
//
// It is padded rather than taken as it comes because a chain keeps its keys at a fixed width,
// where two that differ only in leading zeros are one key. Doing the same here is what keeps
// the two from disagreeing about which keys are the same key.
func (e *Evaluator) slot(operand ir.Operand) []byte {
	return byteutil.PaddingTape(e.value(operand), e.tapeSize)
}
