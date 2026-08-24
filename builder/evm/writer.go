package evm

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// WritePush emits PUSH<size> with the operand as a tape. The push opcodes are contiguous
// from PUSH1 to PUSH32, so the opcode is derived from the width.
func WritePush(w io.Writer, operand []byte, size int) (int, error) {
	size = byteutil.TapeSize(size)
	if _, err := w.Write([]byte{OpPush1 + byte(size-1)}); err != nil {
		return 0, err
	}
	if _, err := w.Write(byteutil.PaddingTape(operand, size)); err != nil {
		return 0, err
	}
	return 0, nil
}

// WriteMask cuts the value on top of the stack down to the tape width.
//
// A tape of N bytes holds values modulo 2^(8N) — that is what the evaluator does, keeping the
// last N bytes of every result. The EVM works in words of 32 bytes and wraps at 2^256, so
// without this the same program answers two different things: at one byte, 255 + 1 is 0 in
// the evaluator and 256 on chain.
//
// At the full width the mask is every bit set, so nothing is written: the common case pays
// nothing for this.
func WriteMask(w io.Writer, size int) (int, error) {
	size = byteutil.TapeSize(size)
	if size >= byteutil.MaxTapeSize {
		return 0, nil
	}

	mask := bytes.Repeat([]byte{0xff}, size)
	if _, err := w.Write(append([]byte{OpPush1 + byte(size-1)}, mask...)); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpAnd})
}

// WriteAdd emits ADD. Lowering guarantees [left, right] on stack (LIFO); Builder is mechanical.
func WriteAdd(w io.Writer) (int, error) {
	return w.Write([]byte{OpAdd})
}

func WriteMultiply(w io.Writer) (int, error) {
	return w.Write([]byte{OpMul})
}

func WriteSubtract(w io.Writer) (int, error) {
	return w.Write([]byte{OpSub})
}

func WriteDivide(w io.Writer) (int, error) {
	return w.Write([]byte{OpDiv})
}

// WriteSave puts a value on the EVM stack as a tape of the configured width. There is no
// special case for booleans any more: they are tapes like everything else.
func WriteSave(w io.Writer, left []byte, size int) (int, error) {
	return WritePush(w, left, size)
}

// WriteIdent stores a value under a name, in the place the frame keeps for it.
//
// The address goes in two bytes. It used to go in one, and a slot is thirty-two wide, so the
// ninth name in a contract was given the address of the first: two names became one piece of
// memory, and each wrote over the other.
func WriteIdent(w io.Writer, names map[string]int, ident []byte) (int, error) {
	if err := WriteFrameAddress(w, names[string(ident)]); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpMemoryStore})
}

// WriteLoad reads what a name holds.
func WriteLoad(w io.Writer, names map[string]int, ident []byte) (int, error) {
	if err := WriteFrameAddress(w, names[string(ident)]); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpMemoryLoad})
}

// WriteAnswer hands the value on the stack to the chain.
//
// It goes through a slot of its own rather than through the first one, which holds where the
// running scope's frame starts: writing the answer there would lose the frame in the act of
// answering.
func WriteAnswer(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpPush1, RETURN_SCRATCH}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpPush1, 0x20, OpPush1, RETURN_SCRATCH, OpReturn})
}

// WriteReturn ends a scope: the value it answers with is on the stack, and the address to go
// back to is under it.
//
// It goes back to whoever called and never to the chain, even when the caller is the way in
// from a transaction. That is what lets one body serve both: a scope called by another has to
// hand its value over and let that one carry on, and only the way in from a transaction ends
// the call — which it does in the epilogue, where it belongs.
func WriteReturn(w io.Writer) (int, error) {
	return w.Write([]byte{OpSwap1, OpJump})
}

// WriteFrameAddress puts on the stack where something at this offset of the frame lives.
//
// A scope names its places by how far into its frame they are, and where the frame starts is
// read at the moment it is used — which is what makes two activations of one scope keep their
// own, and what a call will move.
func WriteFrameAddress(w io.Writer, offset int) error {
	if _, err := WritePush2(w, offset); err != nil {
		return err
	}
	if _, err := w.Write([]byte{OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd}); err != nil {
		return err
	}
	return nil
}

// WriteScopePrologue emits the way in from a transaction.
//
// A scope can be entered two ways, and they differ in exactly one thing: where the values
// applied to it come from. A transaction brings them in the calldata; another scope has them
// as values it just worked out. Everything after that is the same scope doing the same thing,
// so it is written once and entered twice:
//
//	external:  JUMPDEST          <- the dispatcher jumps here
//	           <prologue>        copies the calldata into the frame
//	           PUSH2 <epilogue>  the address to come back to
//	           PUSH2 <internal>
//	           JUMP
//	epilogue:  JUMPDEST          <- the body comes back here, value on the stack
//	           <answer the chain>
//	internal:  JUMPDEST          <- another scope jumps here, having written the frame
//	           <body>            feed(n) reads the frame, whoever filled it
//	           SWAP1; JUMP       back to whoever called
//
// The prologue is what makes the body have one shape. Without it the body would have to know
// how it was entered — reading the calldata one way and the frame the other — and every
// instruction that touches an applied value would carry that question. With it, the body
// reads the frame and nothing else, and the two ways in differ only in who filled it.
//
// It copies as many positions as the body addresses, which is the highest it feeds plus one.
// A position the calldata does not reach reads zero, which is what the language answers for
// reading past what was applied: CALLDATALOAD past the end gives zeros, so the rule costs
// nothing to keep.
//
// Each value is cut to the tape width on the way in rather than on every read. An argument
// arrives as a thirty-two byte word whatever the tape is, and letting it through whole would
// hand a contract a value its own language cannot hold.
//
// The body answers to whoever called it and never to the chain, even when the caller is the
// prologue: answering the chain is what the epilogue does, and keeping it there is what lets
// the same body serve a transaction and a scope.
func WriteScopePrologue(w io.Writer, reads int, tapeSize int, epilogue, internal int) error {
	for at := 0; at < reads; at++ {
		if _, err := w.Write([]byte{OpPush1, GetCalldataArgsOffset(uint64(at))}); err != nil {
			return err
		}
		if _, err := w.Write([]byte{OpCallDataLoad}); err != nil {
			return err
		}
		if _, err := WriteMask(w, tapeSize); err != nil {
			return err
		}
		if err := WriteFrameAddress(w, at*MEMORY_SLOT_SIZE); err != nil {
			return err
		}
		if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
			return err
		}
	}

	if _, err := WritePush2(w, epilogue); err != nil {
		return err
	}
	if _, err := WritePush2(w, internal); err != nil {
		return err
	}
	_, err := w.Write([]byte{OpJump})
	return err
}

// WriteScopeEpilogue emits what a transaction's way in comes back to: the value the scope
// answered with is on the stack, and it goes to the chain.
//
// It is here and not at the end of the body because the body does not know who called it. A
// scope called by another has to hand its value back and let that one carry on; only the way
// in from a transaction ends the call.
func WriteScopeEpilogue(w io.Writer) error {
	_, err := WriteAnswer(w)
	return err
}

// WriteSignificantLength answers, in bits, how much of the value on the stack is significant:
// eight times the number of bytes it takes once the zeros in front of it are dropped, and
// eight for a value that is nothing but zeros.
//
// Every tape operation is defined in terms of that number, because a tape is a shift register
// and what enters it is the value's own bytes and not a fixed run. Off a chain it is the
// length of a slice. On one there is nothing to ask: a value is a word, and how much of it
// means something has to be worked out.
//
// It is worked out by halving. Five times over, the value is asked whether anything survives a
// shift of sixteen bytes, then eight, then four, then two, then one; each answer is one or
// zero, so multiplying the shift by it either takes that step or takes nothing, and the same
// product is added to the running total. Nothing branches, which is what keeps this a run of
// instructions rather than five jumps, and the total is exact for every word.
//
// The last byte is added at the end and is never asked about, which is what makes a value of
// zero one byte long rather than none — the same answer the evaluator gives, and the one that
// keeps "pull t 0" a shift by one place.
func WriteSignificantLength(w io.Writer) error {
	// The running total starts at nothing and goes under the value, which is where every step
	// leaves it.
	if _, err := w.Write([]byte{OpPush1, 0x00, OpSwap1}); err != nil {
		return err
	}

	for _, step := range []int{16, 8, 4, 2, 1} {
		bits := byte(step * BYTE_SIZE)
		// The value is on top and the total under it, and both come back that way.
		if _, err := w.Write([]byte{
			OpDup1, OpPush1, bits, OpShiftRight, OpIsZero, OpIsZero, OpPush1, bits, OpMul,
			OpDup1, OpSwap2, OpSwap1, OpShiftRight, OpSwap2, OpAdd, OpSwap1,
		}); err != nil {
			return err
		}
	}

	// What is left of the value is under a byte, and that byte counts.
	_, err := w.Write([]byte{OpPop, OpPush1, BYTE_SIZE, OpAdd})
	return err
}

// staticLength answers how many bytes of an operand mean something, when that can be known
// while compiling. It can when the program wrote the value down; it cannot when another
// instruction works it out.
func staticLength(operand ir.Operand, tapeSize int) (int, bool) {
	if operand.Kind() != ir.KindImm {
		return 0, false
	}
	return len(byteutil.ExtractSignificantBytes(byteutil.PaddingTape(operand.Bytes(), tapeSize))), true
}

// WriteTapePull shifts a tape left and lets values in at the right, one after another, keeping
// the tape's width — so whatever reaches the left end falls off.
//
// Each value enters as its own significant bytes, so where the tape has to move by is how long
// the value is. When the program wrote the values down, that is known while compiling and the
// whole run of them collapses into one shift and one or: a tape literal costs two instructions
// on a chain, however many values were written between the brackets.
//
// When another instruction works the value out, the length is worked out beside it. That is
// the ordinary "pull t x", and it is written for one value: a literal built out of values a
// program computes would need each of them brought to the top of the stack in turn, and that
// is refused rather than written wrong.
func WriteTapePull(w io.Writer, inst ir.Instruction, tapeSize int) error {
	values := valueOperands(inst)
	tape, items := values[0], values[1:]

	// The tape is on the stack when another instruction worked it out, and reaches it here
	// when the program wrote it down.
	if values[0].Kind() == ir.KindImm {
		if _, err := WritePush(w, values[0].Bytes(), tapeSize); err != nil {
			return err
		}
	}

	entering := make([]byte, 0, len(items)*tapeSize)
	shift := 0
	for _, item := range items {
		length, known := staticLength(item, tapeSize)
		if !known {
			return writeTapePullOver(w, tape, items, tapeSize)
		}
		entering = append(entering, byteutil.PaddingTape(item.Bytes(), tapeSize)[tapeSize-length:]...)
		shift += length * BYTE_SIZE
	}

	if err := writeShiftBy(w, shift, OpShiftLeft); err != nil {
		return err
	}
	if len(entering) > 0 {
		if _, err := WritePush(w, byteutil.PaddingTape(entering, byteutil.TapeSize(len(entering))), len(entering)); err != nil {
			return err
		}
		if _, err := w.Write([]byte{OpOr}); err != nil {
			return err
		}
	}
	_, err := WriteMask(w, tapeSize)
	return err
}

// writeTapePullOver pulls one value a program works out, which is the only shape of that the
// builder writes. The tape is under it on the stack and both are needed, so the length is
// taken off a copy and the two change places around it.
func writeTapePullOver(w io.Writer, tape ir.Operand, items []ir.Operand, tapeSize int) error {
	if len(items) != 1 {
		return fmt.Errorf(
			"a tape built from %d values a program works out does not reach the bytecode yet: write them down, or pull them one at a time",
			len(items))
	}
	if tape.Kind() != ir.KindRef {
		return fmt.Errorf("pulling onto a tape a program works out is not written yet")
	}

	if _, err := w.Write([]byte{OpDup1}); err != nil {
		return err
	}
	if err := WriteSignificantLength(w); err != nil {
		return err
	}
	// The length is on top, the value under it and the tape under that. The tape has to be on
	// top for the shift, and the length above it.
	if _, err := w.Write([]byte{OpSwap2, OpSwap1, OpSwap2, OpShiftLeft, OpOr}); err != nil {
		return err
	}
	_, err := WriteMask(w, tapeSize)
	return err
}

// WriteTapePush shifts a tape right and lets a value in at the left, keeping the tape's width —
// so whatever reaches the right end falls off.
//
// The value goes in front of the tape and the first tape-width of bytes is what is kept, which
// on a chain is the value moved up by what is left of the tape's width and the tape moved down
// by the value's own. A value is a tape, so it is never wider than one, and neither shift can
// run backwards.
func WriteTapePush(w io.Writer, inst ir.Instruction, tapeSize int) error {
	values := valueOperands(inst)
	item := values[1]

	// The tape is on the stack when another instruction worked it out, and reaches it here
	// when the program wrote it down.
	if values[0].Kind() == ir.KindImm {
		if _, err := WritePush(w, values[0].Bytes(), tapeSize); err != nil {
			return err
		}
	}

	if length, known := staticLength(item, tapeSize); known {
		if err := writeShiftBy(w, length*BYTE_SIZE, OpShiftRight); err != nil {
			return err
		}
		entering := byteutil.PaddingTape(item.Bytes(), tapeSize)[tapeSize-length:]
		if err := writeShiftedIn(w, entering, (tapeSize-length)*BYTE_SIZE); err != nil {
			return err
		}
		_, err := WriteMask(w, tapeSize)
		return err
	}

	if _, err := w.Write([]byte{OpDup1}); err != nil {
		return err
	}
	if err := WriteSignificantLength(w); err != nil {
		return err
	}
	// The tape moves down by the length, keeping a copy of the length for the other shift.
	if _, err := w.Write([]byte{OpSwap2, OpDup3, OpShiftRight, OpSwap2}); err != nil {
		return err
	}
	if _, err := WritePush2(w, tapeSize*BYTE_SIZE); err != nil {
		return err
	}
	if _, err := w.Write([]byte{OpSub, OpShiftLeft, OpOr}); err != nil {
		return err
	}
	_, err := WriteMask(w, tapeSize)
	return err
}

// writeShiftedIn ors a value into the tape at a place known while compiling.
func writeShiftedIn(w io.Writer, entering []byte, shift int) error {
	if len(entering) == 0 {
		return nil
	}
	value := new(big.Int).SetBytes(entering)
	value.Lsh(value, uint(shift))
	if _, err := WritePush(w, byteutil.PaddingTape(value.Bytes(), byteutil.MaxTapeSize), byteutil.MaxTapeSize); err != nil {
		return err
	}
	_, err := w.Write([]byte{OpOr})
	return err
}

// writeShiftBy moves what is on the stack by a number of bits known while compiling. A shift
// of nothing is written as nothing.
func writeShiftBy(w io.Writer, bits int, op byte) error {
	if bits == 0 {
		return nil
	}
	if bits > byteutil.MaxTapeSize*BYTE_SIZE {
		bits = byteutil.MaxTapeSize * BYTE_SIZE
	}
	if _, err := WritePush2(w, bits); err != nil {
		return err
	}
	_, err := w.Write([]byte{op})
	return err
}

// WriteTapeHead keeps the first bytes of what a tape says, and WriteTapeTail drops them.
//
// Both count in significant bytes, so both start by asking how long the value is. What is kept
// is measured from the other end: keeping the first n of s bytes means moving the tape down by
// s minus n, and dropping them means keeping that many. The index is taken modulo the tape
// width, so it can never be out of bounds, and it is held to the value's own length — asking
// for more of a tape than it has gives all of it.
func WriteTapeHead(w io.Writer, inst ir.Instruction, tapeSize int) error {
	if err := writeRemainingLength(w, inst, tapeSize); err != nil {
		return err
	}
	_, err := w.Write([]byte{OpShiftRight})
	return err
}

func WriteTapeTail(w io.Writer, inst ir.Instruction, tapeSize int) error {
	if err := writeRemainingLength(w, inst, tapeSize); err != nil {
		return err
	}
	// Every bit below what is kept, which for a whole word comes out of the shift as zero and
	// out of the subtraction as all ones — which is the tape untouched, and is what dropping
	// none of it means.
	_, err := w.Write([]byte{OpPush1, 0x01, OpSwap1, OpShiftLeft, OpPush1, 0x01, OpSwap1, OpSub, OpAnd})
	return err
}

// writeRemainingLength leaves, above the tape, how many bits of it are past the index.
func writeRemainingLength(w io.Writer, inst ir.Instruction, tapeSize int) error {
	at := int(byteutil.ToUint64(inst.GetRight().Bytes())) % tapeSize

	if _, err := w.Write([]byte{OpDup1}); err != nil {
		return err
	}
	if err := WriteSignificantLength(w); err != nil {
		return err
	}
	if at == 0 {
		return nil
	}

	// The index can be past the end of what the value says, and then there is nothing past it:
	// the subtraction would run backwards, so it is multiplied by whether it should have
	// happened at all.
	bits := byte(at * BYTE_SIZE)
	_, err := w.Write([]byte{
		OpDup1, OpPush1, bits, OpGreaterThan, OpIsZero,
		OpSwap1, OpPush1, bits, OpSwap1, OpSub, OpMul,
	})
	return err
}

// WriteJoin lays tapes end to end into one value.
//
// A run has no header, no length and no tag — it is the tapes and nothing else, which is what
// the language says a shape is. So on a chain it is a word like every other value, with the
// first tape at the far end and the last one at the bottom, which is where a tape sits inside
// a word anyway. Point{10, 20} at a tape of eight bytes is 10 << 64 | 20, the same number the
// evaluator answers, so a scope answering a run answers the same thing on both sides.
//
// That is also the ceiling: a word is thirty-two bytes and a run is as many tapes as it has,
// so a shape of five fields at the default tape does not fit, and the builder refuses it
// rather than writing a value with the first field shifted off the end.
//
// It is built from the last tape back, because that is the order the stack offers them. The
// ones another instruction worked out are on it already, put there in field order by the
// lowering, so the last of them is on top; the ones the program wrote down reach the stack
// here. Either way what has been built so far sits under what comes next, which is why a tape
// taken off the stack changes places with it and one pushed does not.
func WriteJoin(w io.Writer, inst ir.Instruction, tapeSize int) error {
	tapes := valueOperands(inst)
	if run := len(tapes) * tapeSize; run > byteutil.MaxTapeSize {
		return fmt.Errorf(
			"a run of %d tapes is %d bytes and a word is %d: a shape this wide does not reach the bytecode",
			len(tapes), run, byteutil.MaxTapeSize)
	}

	for at := len(tapes) - 1; at >= 0; at-- {
		tape := tapes[at]
		switch {
		case tape.Kind() == ir.KindImm:
			if _, err := WritePush(w, tape.Bytes(), tapeSize); err != nil {
				return err
			}
		case at < len(tapes)-1:
			// It is under what has been built so far, and it is what the shift applies to.
			if _, err := w.Write([]byte{OpSwap1}); err != nil {
				return err
			}
		}

		// A tape wider than the tape size is cut to it, which is what the evaluator does
		// when it lays one into a run: a run of tapes holds tapes.
		if _, err := WriteMask(w, tapeSize); err != nil {
			return err
		}

		if bits := (len(tapes) - 1 - at) * tapeSize * BYTE_SIZE; bits > 0 {
			if _, err := WritePush(w, byteutil.FromUint64(uint64(bits)), 1); err != nil {
				return err
			}
			if _, err := w.Write([]byte{OpShiftLeft}); err != nil {
				return err
			}
		}

		if at < len(tapes)-1 {
			if _, err := w.Write([]byte{OpOr}); err != nil {
				return err
			}
		}
	}

	return nil
}

// WriteField takes one tape out of a run.
//
// The run is on the stack and the two numbers are the instruction's own: which tape, and how
// many the run has. Both are needed because a run is counted from its first tape and kept from
// its last — nothing in the value says where it ends, so the only way to the tape at index i
// is to know there are n of them and count back.
func WriteField(w io.Writer, inst ir.Instruction, tapeSize int) error {
	operands := inst.GetOperands()
	at := int(byteutil.ToUint64(operands[1].Bytes()))
	tapes := int(byteutil.ToUint64(operands[2].Bytes()))

	if bits := (tapes - 1 - at) * tapeSize * BYTE_SIZE; bits > 0 {
		if _, err := WritePush(w, byteutil.FromUint64(uint64(bits)), 1); err != nil {
			return err
		}
		if _, err := w.Write([]byte{OpShiftRight}); err != nil {
			return err
		}
	}

	_, err := WriteMask(w, tapeSize)
	return err
}

// WriteCall enters another scope of this contract and comes back with what it answered.
//
// It is a jump, and deliberately not a message call. A message call to your own contract is a
// transaction against yourself: a frame of execution of its own, its own gas stipend, its own
// view of storage, and an answer that comes back as returndata to be copied out. None of that
// is what one scope calling another means, and paying for it would make a call inside a
// program cost what a call between contracts costs.
//
// What a jump does not bring is anywhere for the callee to keep the values applied to it, and
// that is what the frame is for. The callee's goes right after the caller's, so the two never
// share a slot — which is also what lets a scope be entered while it is already running.
//
//	<put each applied value in the callee's frame>
//	<move the frame pointer past this scope's frame>
//	PUSH2 <back>        where to carry on once the callee answers
//	PUSH2 <internal>    the callee, entered past its prologue: its frame is written already
//	JUMP
//	back: JUMPDEST
//	<move the frame pointer back>
//
// The applied values reach the frame from two places, and the order matters. One the program
// wrote down, and this is where it reaches the stack at all. One another instruction worked
// out, and it is on the stack already — the lowering put its producer right in front of this
// call. So the written-down ones are stored first, each pushed and put away in one breath, and
// the worked-out ones are taken off the top afterwards, in reverse of the order they went on.
//
// The pointer is moved back by subtracting what moved it forward rather than by keeping a
// copy, because a copy would have to be kept somewhere across the call — and the only
// somewhere is the stack, under an answer the callee's own work is piled on top of.
func WriteCall(w io.Writer, inst ir.Instruction, scope Scope, at int) error {
	name := string(inst.GetLeft().Bytes())
	callee, known := scope.Entries[name]
	if !known {
		return fmt.Errorf(
			"a call to %q cannot be written: only a scope bound at the top of a program reaches the bytecode", name)
	}

	// Where to carry on is where this call ends, which is known by writing it: every push is
	// a fixed size, so what is measured with a wrong address is what is written with the
	// right one.
	var measured counter
	if err := writeCallOut(&measured, inst, scope, 0, callee); err != nil {
		return err
	}
	if err := writeCallOut(w, inst, scope, at+int(measured), callee); err != nil {
		return err
	}

	if _, err := w.Write([]byte{OpJumpDestiny}); err != nil {
		return err
	}
	return writeFrameMove(w, scope.Frame, OpSub)
}

// writeCallOut fills the callee's frame, moves the pointer onto it, and goes.
func writeCallOut(w io.Writer, inst ir.Instruction, scope Scope, back int, callee Entry) error {
	applied := valueOperands(inst)

	for at, operand := range applied {
		if operand.Kind() != ir.KindImm {
			continue
		}
		if _, err := WritePush(w, operand.Bytes(), scope.TapeSize); err != nil {
			return err
		}
		if err := writeFrameStore(w, scope.Frame+at*MEMORY_SLOT_SIZE); err != nil {
			return err
		}
	}

	for at := len(applied) - 1; at >= 0; at-- {
		if applied[at].Kind() != ir.KindRef {
			continue
		}
		if err := writeFrameStore(w, scope.Frame+at*MEMORY_SLOT_SIZE); err != nil {
			return err
		}
	}

	// A position the callee reads and this call did not fill has to answer with zeros, which
	// is what the language answers for reading past what was applied. On the way in from a
	// transaction that is free — the calldata gives zeros past its end — but a frame is
	// memory an earlier activation already used, so what is left there is the last call's
	// values. Two scopes reading two positions, one of them called with one, and the second
	// read the first call's second value.
	for at := len(applied); at < callee.Reads; at++ {
		if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil {
			return err
		}
		if err := writeFrameStore(w, scope.Frame+at*MEMORY_SLOT_SIZE); err != nil {
			return err
		}
	}

	if err := writeFrameMove(w, scope.Frame, OpAdd); err != nil {
		return err
	}
	if _, err := WritePush2(w, back); err != nil {
		return err
	}
	if _, err := WritePush2(w, callee.At); err != nil {
		return err
	}
	_, err := w.Write([]byte{OpJump})
	return err
}

// writeFrameStore puts the value on top of the stack at this offset of the frame.
func writeFrameStore(w io.Writer, offset int) error {
	if err := WriteFrameAddress(w, offset); err != nil {
		return err
	}
	_, err := w.Write([]byte{OpMemoryStore})
	return err
}

// writeFrameMove moves the frame pointer by a number of bytes, onto the callee's frame with
// ADD and back off it with SUB. Both read the pointer as it is now, so nested calls stack.
func writeFrameMove(w io.Writer, by int, op byte) error {
	if _, err := WritePush2(w, by); err != nil {
		return err
	}
	if _, err := w.Write([]byte{OpPush1, FRAME_POINTER, OpMemoryLoad, op}); err != nil {
		return err
	}
	if _, err := w.Write([]byte{OpPush1, FRAME_POINTER, OpMemoryStore}); err != nil {
		return err
	}
	return nil
}

// ScopeInternalAt answers the byte another scope enters this one at: past the way in from a
// transaction, and past what that way comes back to.
//
// It is measured rather than added up from a table of sizes, for the reason every address here
// is: a table is a second description of the same bytes, and two descriptions drift.
func ScopeInternalAt(base, params, tapeSize int) (int, error) {
	var measured counter
	if err := WriteScopePrologue(&measured, params, tapeSize, 0, 0); err != nil {
		return 0, err
	}
	// The way in opens with a JUMPDEST the dispatcher lands on; the prologue follows; what it
	// comes back to opens with one of its own, and the way in from another scope with a third.
	return base + 1 + int(measured) + 1 + ANSWER_SIZE, nil
}

// WriteScope emits a scope whole: the way in from a transaction, what that way comes back to,
// and the blocks both ways share.
//
// The addresses inside it are worked out by measuring the prologue, which is the only part
// whose length depends on the scope — one copy per value applied to it. Every push is a fixed
// size, so measuring once is enough.
func WriteScope(bs io.Writer, blocks []ir.Block, entry ir.BlockID, tapeSize, base int, entries map[string]Entry) error {
	order := layoutOf(blocks, entry)
	scope := ScopeOf(blocks, order, tapeSize, entries, false)

	internal, err := ScopeInternalAt(base, scope.Params, tapeSize)
	if err != nil {
		return err
	}
	epilogue := internal - 1 - ANSWER_SIZE

	if _, err := bs.Write([]byte{OpJumpDestiny}); err != nil {
		return err
	}
	if err := WriteScopePrologue(bs, scope.Params, tapeSize, epilogue, internal); err != nil {
		return err
	}
	if _, err := bs.Write([]byte{OpJumpDestiny}); err != nil {
		return err
	}
	if err := WriteScopeEpilogue(bs); err != nil {
		return err
	}

	// The first block opens with a JUMPDEST of its own, which is the way in from another
	// scope.
	return writeBlocks(bs, blocks, order, internal, scope)
}

// WriteFrameStart writes where the first frame begins, which every scope reads from until a
// call moves it.
func WriteFrameStart(w io.Writer) error {
	if _, err := WritePush2(w, FRAME_BASE); err != nil {
		return err
	}
	if _, err := w.Write([]byte{OpPush1, FRAME_POINTER, OpMemoryStore}); err != nil {
		return err
	}
	return nil
}

// WriteGetArg reads one of the values applied to the running scope.
//
// It reads the frame, and never the calldata. Whoever entered the scope put them there: the
// way in from a transaction copies them out of the calldata, and a scope calling another
// writes the values it worked out. That is the whole of what the frame buys — a body that does
// not know how it was entered.
//
// The narrowing to the tape width happened where the value came in, so there is none here.
func WriteGetArg(w io.Writer, left []byte, size int) (int, error) {
	at := int(byteutil.ToUint64(left))
	if err := WriteFrameAddress(w, at*MEMORY_SLOT_SIZE); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpMemoryLoad})
}

func WriteStop(w io.Writer) (int, error) {
	return w.Write([]byte{OpStop})
}

// WritePush2 emits PUSH2 with a number in two bytes, big-endian.
//
// Two bytes because one is not enough and three would never be: a published contract cannot
// pass 24,576 bytes (EIP-170), so every length and every address inside a legal one fits in
// 65,535. There is no size after this one.
func WritePush2(w io.Writer, n int) (int, error) {
	return w.Write([]byte{OpPush2, byte(n >> 8), byte(n)})
}

// WriteInstantiateBlock emits the constructor: it copies the runtime out of the code being
// deployed and hands it to the chain, which is what the chain then keeps.
//
// The size is pushed in two bytes. It used to be one, and a runtime past 255 bytes was
// truncated by the conversion — a program with three deferred scopes reached that — so the
// constructor asked for 96 bytes of a contract that had 352. It deployed, and what the chain
// kept was cut off in the middle of an instruction.
//
// The offset it copies from is this block's own length, since the runtime begins right after
// it. That number is derived rather than written down twice: the two used to be the same
// literal in two places, and changing a push meant remembering both.
func WriteInstantiateBlock(w io.Writer, runtimeSize int) (int, error) {
	if _, err := WritePush2(w, runtimeSize); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, byte(INSTANTIATE_BLOCK_SIZE)}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpCodeCopy}); err != nil {
		return 0, err
	}
	if _, err := WritePush2(w, runtimeSize); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpReturn})
}

func WriteNoMatchDispatcher(w io.Writer) (int, error) {
	return w.Write([]byte{OpStop})
}

// WriteDispatcher emits the entry for one scope: it reads the selector out of the calldata,
// compares it with the one this scope answers to, and jumps to the body when they match.
//
// The address goes in two bytes. It used to go in one, so a body living past byte 255 of the
// runtime was jumped to at an address that had been truncated — a contract with twelve scopes
// answered for the first and refused the third with "invalid jump destination". The body was
// there; the dispatcher could not name it.
func WriteDispatcher(bs io.Writer, id string, jumpTo int) (int, error) {
	if _, err := bs.Write([]byte{OpPush1, 0x00}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpCallDataLoad}); err != nil {
		return 0, err
	}
	// Isolate the first 4 bytes of the keccak256 hash of the id
	if _, err := bs.Write([]byte{OpPush1, byte((CALLDATA_SLOT_READABLE - 4) * BYTE_SIZE)}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpShiftRight}); err != nil {
		return 0, err
	}
	selector := crypto.Keccak256([]byte(id))[:4]
	if _, err := bs.Write(append([]byte{OpPush4}, selector...)); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpEqual}); err != nil {
		return 0, err
	}
	if _, err := WritePush2(bs, jumpTo); err != nil {
		return 0, err
	}
	return bs.Write([]byte{OpJumpIf})
}

// WriteDispatchers emits what every call arrives at: where the first frame begins, and then
// one entry per scope.
//
// The frame is set here rather than in the constructor because the constructor does not run
// with the contract — it runs once, to deploy it, and what it writes to memory then is gone.
// Every call starts from nothing and says where its frame is.
func WriteDispatchers(bs io.Writer, ds []Dispatcher) (int, error) {
	if err := WriteFrameStart(bs); err != nil {
		return 0, err
	}

	dispatcherLen := DISPATCHER_BYTES_SIZE * len(ds)
	// After dispatchers we have the no-match dispatcher (STOP); referenced code starts after
	// it — and all of it after the frame.
	referencedStart := FRAME_START_SIZE + dispatcherLen + NO_MATCH_DISPATCHER_SIZE

	for _, d := range ds {
		jumpTo := referencedStart + d.Offset
		if _, err := WriteDispatcher(bs, string(d.Selector), jumpTo); err != nil {
			return 0, err
		}
	}

	// No-match STOP only when we have selectors; otherwise runtime starts with root code.
	if len(ds) > 0 {
		if _, err := WriteNoMatchDispatcher(bs); err != nil {
			return 0, err
		}
		return FRAME_START_SIZE + dispatcherLen + NO_MATCH_DISPATCHER_SIZE, nil
	}

	return FRAME_START_SIZE + dispatcherLen, nil
}

func WriteBodyCode(bs io.Writer, ds []Dispatcher, root *bytes.Buffer) (int, error) {
	for _, d := range ds {
		if _, err := bs.Write(d.Code.Bytes()); err != nil {
			return 0, err
		}
	}
	if root != nil {
		if _, err := bs.Write(root.Bytes()); err != nil {
			return 0, err
		}
	}
	return 0, nil
}

// comparisons maps each of them to what the EVM makes of it.
//
// The two that are not symmetric are read top-first, which the lowering already knows: `a
// bigger b` arrives with a on top, and GT reads the top as the left-hand side.
//
// "different" is equality turned over, since the EVM has no opcode for it.
//
// And and or are the logical ones and not the bitwise ones — Aurora asks whether both values
// hold, not which bits they share, so `2 and 1` is true where a bitwise AND would answer zero.
// OR of the raw values is already non-zero when either is, so it only needs narrowing to one
// or zero; AND cannot be done that way, and is written as the other one turned inside out:
// not (not a or not b).
var comparisons = map[byte][]byte{
	ir.OpEquals:  {OpEqual},
	ir.OpDiff:    {OpEqual, OpIsZero},
	ir.OpBigger:  {OpGreaterThan},
	ir.OpSmaller: {OpLessThan},
	ir.OpOr:      {OpOr, OpIsZero, OpIsZero},
	ir.OpAnd:     {OpIsZero, OpSwap1, OpIsZero, OpOr, OpIsZero},
}

// passesThrough names the instructions that write nothing and are worth what they were given.
//
// A print is one. The log has nowhere to go on a chain — that is a decision, not a gap — but a
// print is an expression like any other and is worth the value it showed, which is what lets it
// be written into a program that is already working without changing what that program answers.
// So on a chain it is the identity: nothing is written, and the value carries on.
//
// It carried on by accident before, and only sometimes. A value another instruction worked out
// was already on the stack, so a print over it happened to leave the right thing there; a value
// the program wrote down was never put on the stack at all, and the contract underflowed the
// first time anything read what the print was worth.
var passesThrough = map[byte]bool{
	ir.OpPrintBytes:   true,
	ir.OpPrintChars:   true,
	ir.OpPrintDecimal: true,
}

// ordersItsOwn names the instructions that put their own written-down values on the stack.
//
// Most take a pair and the pair is enough for WriteImmediates to know where each one goes. An
// instruction that takes as many values as it was given is not: which of them are on the stack
// already and which are being written down is what decides the order, and only the instruction
// knows what it does with them. A save is here for the older reason — it is the value.
var ordersItsOwn = map[byte]bool{
	ir.OpSave: true,
	ir.OpCall: true,
	ir.OpJoin: true,
	ir.OpPull: true,
	ir.OpPush: true,
}

// WriteImmediates puts on the stack the values an instruction carries inside itself.
//
// A Ref was already put there by whoever produced it, and the lowering saw to it that it
// landed in the right place. An Imm has no producer — the program wrote the value down — so
// the instruction that takes it is where it reaches the stack.
//
// Which is why there is a SWAP1 here. When an immediate has to sit *under* a value that is
// already on top, pushing it puts it on the wrong side, and the two have to change places. It
// is the one case, and it is written out rather than inferred: for a subtraction the EVM
// computes top minus next, so "t - 2" needs the two underneath and "2 - t" does not.
func WriteImmediates(w io.Writer, inst ir.Instruction, tapeSize int) error {
	values := valueOperands(inst)
	for at, operand := range values {
		if operand.Kind() != ir.KindImm {
			continue
		}
		if _, err := WritePush(w, operand.Bytes(), tapeSize); err != nil {
			return err
		}
		// It is the first of two and the other is already on the stack, so it landed on
		// top of what it belongs under.
		if at == 0 && len(values) == 2 && values[1].Kind() == ir.KindRef {
			if _, err := w.Write([]byte{OpSwap1}); err != nil {
				return err
			}
		}
	}
	return nil
}

// counter is a writer that keeps how much was written to it and nothing else.
type counter int

func (c *counter) Write(p []byte) (int, error) {
	*c += counter(len(p))
	return len(p), nil
}

// A Scope is what writing an instruction needs to know beyond the instruction itself: the
// facts that belong to the scope it is written inside rather than to the instruction written.
//
// All of it is known before a byte is written, which is what blocks bought. The places a scope
// keeps used to be handed out as the writer went — a name got the next free slot at the moment
// its binding was written — so writing the same scope twice, which measuring does, had to be
// careful to hand out the same places both times. Now the frame is laid out once and read.
type Scope struct {
	// TapeSize is how wide a value is, which every result is cut back to.
	TapeSize int
	// Answers says whether ending the scope ends the call rather than handing the value back.
	// Code no scope holds — the top of a program — has nobody to go back to.
	Answers bool
	// Params is how many values are applied to this scope, which is what its first block
	// takes and what its prologue copies.
	Params int
	// Names answers where in the frame each name this scope binds is kept.
	Names map[string]int
	// Frame is how much memory this scope keeps: the values applied to it, then its names. A
	// call puts the frame of whoever it calls right after this one.
	Frame int
	// Entries answers, for the name a scope was bound to, what a call needs to know about it.
	Entries map[string]Entry
}

// An Entry is what a call needs to know about the scope it calls: where to go in, and how many
// values that scope takes.
//
// A scope takes a fixed number — what its first block says — and that is a property of the
// scope. What a call applies is a property of the call. The two need not agree, and where they
// do not is where the caller has work to do.
type Entry struct {
	// At is the byte another scope enters this one at, past its prologue.
	At int
	// Reads is how many values the scope takes.
	Reads int
}

// ScopeOf answers what the writer needs to know about a run of blocks.
//
// The frame is laid out here: the values applied to the scope first, one slot each, then the
// names it binds in the order they are first bound. Reading the whole scope before writing any
// of it is what makes that possible, and it is the thing an instruction list could not offer —
// there, a name in a block that had not been reached yet was a name nobody knew about.
func ScopeOf(blocks []ir.Block, order []ir.BlockID, tapeSize int, entries map[string]Entry, answers bool) Scope {
	params := 0
	if len(order) > 0 {
		params = len(blocks[order[0]].Params)
	}

	names := make(map[string]int)
	offset := params * MEMORY_SLOT_SIZE
	for _, id := range order {
		for _, inst := range blocks[id].Insts {
			if inst.GetOpCode() != ir.OpIdent {
				continue
			}
			name := string(inst.GetLeft().Bytes())
			if _, bound := names[name]; bound {
				continue
			}
			names[name] = offset
			offset += MEMORY_SLOT_SIZE
		}
	}

	return Scope{
		TapeSize: tapeSize,
		Answers:  answers,
		Params:   params,
		Names:    names,
		Frame:    offset,
		Entries:  entries,
	}
}

// WriteInstruction emits one instruction.
//
// The target is the byte a jump goes to, for the two instructions that jump. It is zero while
// measuring, and measures the same either way, since every push is a fixed size.
//
// Arms answers whether an OpReturn ends a branch rather than a scope. The two are the same
// opcode: one says "the value of this scope" and the other "the value of this arm", and which
// it is can be read — an arm names the OpIf it belongs to, and a scope names the OpBeginScope.
//
// Answers says what the third kind does. Code that no scope holds — the top of a program, run
// when a contract has no scope to dispatch to — has nobody to go back to, so its return ends
// the call rather than handing a value over.
func WriteInstruction(bs io.Writer, names map[string]int, inst ir.Instruction, scope Scope, at int) error {
	op := inst.GetOpCode()

	if (handled[op] || passesThrough[op]) && !ordersItsOwn[op] {
		if err := WriteImmediates(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpAdd {
		if _, err := WriteAdd(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpMultiply {
		if _, err := WriteMultiply(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpSubtract {
		if _, err := WriteSubtract(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpDivide {
		if _, err := WriteDivide(bs); err != nil {
			return err
		}
	}

	if op == ir.OpSave {
		if _, err := WriteSave(bs, inst.GetLeft().Bytes(), scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpIdent {
		// A name bound to a scope holds the neutral value here. A scope is not a value on a
		// chain: it is code, reached by a transaction naming it or by another scope calling
		// it, and neither of those goes through the name's place in memory.
		if inst.GetRight().Kind() == ir.KindBlock {
			if _, err := WritePush(bs, nil, scope.TapeSize); err != nil {
				return err
			}
		}
		if _, err := WriteIdent(bs, names, inst.GetLeft().Bytes()); err != nil {
			return err
		}
	}

	if op == ir.OpLoad {
		if _, err := WriteLoad(bs, names, inst.GetLeft().Bytes()); err != nil {
			return err
		}
	}

	if op == ir.OpCall {
		if err := WriteCall(bs, inst, scope, at); err != nil {
			return err
		}
	}

	if op == ir.OpPull {
		if err := WriteTapePull(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpPush {
		if err := WriteTapePush(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpHead {
		if err := WriteTapeHead(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpTail {
		if err := WriteTapeTail(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpJoin {
		if err := WriteJoin(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpField {
		if err := WriteField(bs, inst, scope.TapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpGetFeed {
		if _, err := WriteGetArg(bs, inst.GetLeft().Bytes(), scope.TapeSize); err != nil {
			return err
		}
	}

	// A comparison answers with a tape like any other value, and the EVM already answers
	// these with one or zero, which is what a tape holding true or false is.
	if compare, ok := comparisons[op]; ok {
		if _, err := bs.Write(compare); err != nil {
			return err
		}
	}

	if op == ir.OpExponential {
		if _, err := bs.Write([]byte{OpExp}); err != nil {
			return err
		}
		if _, err := WriteMask(bs, scope.TapeSize); err != nil {
			return err
		}
	}

	return nil
}
