package evaluator

import (
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// What the chain keeps between one transaction and the next, off a chain.
//
// The point of this backend is that the same program answers the same thing on a chain and off
// it, and storage is where that stops being about arithmetic: a program that reads what an
// earlier transaction wrote can only be simulated by something that keeps it. A map does, for
// as long as the evaluator lives, which is one run of a program — a second run starts empty,
// the way a chain does not.
//
// That difference is real and is not hidden: what is simulated here is what one transaction
// sees, which is what a call simulated off a chain is for.

// EvaluateStorageGet answers what is kept under a key.
//
// A key nothing was ever written under answers the neutral tape, which is the same answer
// reading past the end of a run gives, and the same one the chain gives: a slot never written
// is zeros there too.
func (e *Evaluator) EvaluateStorageGet(label []byte, left, right ir.Operand) error {
	kept, written := e.storage[byteutil.ToHex(e.key(left))]
	if !written {
		kept = byteutil.FalseTape(e.tapeSize)
	}
	e.environ.SetTemp(byteutil.ToHex(label), kept)
	return nil
}

// EvaluateStorageSet keeps a value under a key, and leaves the value.
//
// It leaves it because everything in Aurora is an expression and a write is no exception, so
// `s.set("n", 1) + 1` is two — and because a write that answered nothing would be the only
// instruction in the language that does.
func (e *Evaluator) EvaluateStorageSet(label []byte, left, right ir.Operand) error {
	value := byteutil.PaddingTape(e.value(right), e.tapeSize)
	e.storage[byteutil.ToHex(e.key(left))] = value
	e.environ.SetTemp(byteutil.ToHex(label), value)
	return nil
}

// key answers the tape a key is, as wide as every other value.
//
// It is padded rather than taken as it comes because a chain keeps its keys in a fixed width
// and two keys that differ only in leading zeros are one key there. Doing the same here is
// what keeps the two from disagreeing about which values are the same key.
func (e *Evaluator) key(operand ir.Operand) []byte {
	return byteutil.PaddingTape(e.value(operand), e.tapeSize)
}
