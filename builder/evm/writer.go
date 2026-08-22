package evm

import (
	"bytes"
	"io"

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

type IdentOffsetMapper interface {
	GetOffset(ident []byte) int
	SetOffset(ident string, offset int)
	GetLength() uint
	// Base is how far into the frame the names of this scope begin, past the places the
	// values applied to it are kept.
	Base() int
}

// WriteIdent stores a value under a name, in a slot of memory of its own.
//
// The address goes in two bytes. It used to go in one, and a slot is thirty-two wide, so the
// ninth name in a contract was given the address of the first — 8 * 32 is 256, and one byte
// holds none of it. Two names became one piece of memory, and each wrote over the other.
func WriteIdent(w io.Writer, m IdentOffsetMapper, ident []byte) (int, error) {
	offset := m.Base() + int(m.GetLength())*MEMORY_SLOT_SIZE
	if err := WriteFrameAddress(w, offset); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	m.SetOffset(string(ident), offset)
	return 0, nil
}

func WriteLoad(w io.Writer, m IdentOffsetMapper, left []byte) (int, error) {
	if err := WriteFrameAddress(w, m.GetOffset(left)); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpMemoryLoad})
}

// WriteReturn assumes the return value is on the stack (e.g. after ADD). It stores it at
// mem[0] with MSTORE then returns 32 bytes from 0 so RETURN works without prior memory use.
// WriteReturn hands the value on the stack to the chain.
//
// It goes through a slot of its own rather than through the first one, which now holds where
// the running scope's frame starts: writing the answer there would lose the frame in the act
// of answering.
func WriteReturn(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpPush1, RETURN_SCRATCH}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpPush1, 0x20, OpPush1, RETURN_SCRATCH, OpReturn})
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

// FrameNamesAt answers how far into a frame the names of a scope begin.
//
// Before them sit the values applied to the scope, one slot each, and how many of those there
// are is what the body says it reads: the highest position it feeds, plus one. It is the same
// number the compiler warns a short call about, and it is known where the body is written.
func FrameNamesAt(insts []ir.Instruction) int {
	highest := -1
	for _, inst := range insts {
		if inst.GetOpCode() != ir.OpGetFeed {
			continue
		}
		if at := int(byteutil.ToUint64(inst.GetLeft().Bytes())); at > highest {
			highest = at
		}
	}
	return (highest + 1) * MEMORY_SLOT_SIZE
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

// WriteGetArg reads an argument out of the calldata and cuts it to the tape width.
//
// An argument arrives as a 32-byte word whatever the tape is, and the evaluator narrows it to
// a tape on the way in (environ.NewEnviron). Reading the whole word here would let a caller
// hand a contract a value its own language cannot hold.
func WriteGetArg(w io.Writer, left []byte, size int) (int, error) {
	index := byteutil.ToUint64(left)
	offset := GetCalldataArgsOffset(index)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpCallDataLoad}); err != nil {
		return 0, err
	}
	return WriteMask(w, size)
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

// PositionsOf answers the byte each instruction of a scope starts at, and where the scope ends.
//
// It measures by writing — to something that only counts — rather than by adding up a table of
// sizes beside the writer. A table would be a second description of the same thing, and two
// descriptions of one thing drift: this backend has been wrong that way more than once, and
// the drift is silent because the bytes still come out, just at the wrong addresses.
//
// The names are registered into a manager of its own and thrown away, since what is measured
// is how many bytes an instruction takes and every push is a fixed size now.
func PositionsOf(insts []ir.Instruction, tapeSize int, landings map[int]bool, arms map[string]bool) ([]int, error) {
	positions := make([]int, len(insts)+1)
	im := NewIdentManager()
	for at, inst := range insts {
		var measured counter
		if landings[at] {
			measured++
		}
		if err := WriteInstruction(&measured, im, inst, tapeSize, 0, arms); err != nil {
			return nil, err
		}
		positions[at+1] = positions[at] + int(measured)
	}
	return positions, nil
}

// landingsOf answers the instructions a jump arrives at.
//
// The IR counts its jumps in instructions, which is what the evaluator's cursor takes; the EVM
// takes a byte, and it refuses one that is not a JUMPDEST. So the places that are arrived at
// are worked out first, from the counts alone, and each of them opens with one.
//
// A landing one past the last instruction is the way out of an "if" that ends a scope, and it
// gets a JUMPDEST of its own after everything.
func landingsOf(insts []ir.Instruction) map[int]bool {
	landings := make(map[int]bool)
	for at, inst := range insts {
		switch inst.GetOpCode() {
		case ir.OpIf:
			landings[at+1+int(byteutil.ToUint64(inst.GetRight().Bytes()))] = true
		case ir.OpJump:
			landings[at+1+int(byteutil.ToUint64(inst.GetLeft().Bytes()))] = true
		}
	}
	return landings
}

// armsOf answers the labels an "if" answers under, so an OpReturn naming one is known to end a
// branch rather than a scope.
func armsOf(insts []ir.Instruction) map[string]bool {
	arms := make(map[string]bool)
	for _, inst := range insts {
		if inst.GetOpCode() == ir.OpIf {
			arms[byteutil.ToHex(inst.GetLabel())] = true
		}
	}
	return arms
}

// targetOf answers the byte an instruction jumps to, or zero for one that does not jump.
func targetOf(inst ir.Instruction, at int, positions []int) int {
	var ahead int
	switch inst.GetOpCode() {
	case ir.OpIf:
		ahead = int(byteutil.ToUint64(inst.GetRight().Bytes()))
	case ir.OpJump:
		ahead = int(byteutil.ToUint64(inst.GetLeft().Bytes()))
	default:
		return 0
	}
	return positions[at+1+ahead]
}

// WriteInstruction emits one instruction.
//
// The target is the byte a jump goes to, for the two instructions that jump. It is zero while
// measuring, and measures the same either way, since every push is a fixed size.
//
// Arms answers whether an OpReturn ends a branch rather than a scope. The two are the same
// opcode: one says "the value of this scope" and the other "the value of this arm", and which
// it is can be read — an arm names the OpIf it belongs to, and a scope names the OpBeginScope.
func WriteInstruction(bs io.Writer, im *IdentManager, inst ir.Instruction, tapeSize int, target int, arms map[string]bool) error {
	op := inst.GetOpCode()

	if handled[op] && op != ir.OpSave {
		if err := WriteImmediates(bs, inst, tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpAdd {
		if _, err := WriteAdd(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpMultiply {
		if _, err := WriteMultiply(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpSubtract {
		if _, err := WriteSubtract(bs); err != nil {
			return err
		}
		if _, err := WriteMask(bs, tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpDivide {
		if _, err := WriteDivide(bs); err != nil {
			return err
		}
	}

	if op == ir.OpReturn {
		// The value of an arm is already on the stack, which is where whoever is under the
		// branch finds it: there is nothing to write. Only a scope answers to the chain.
		if !arms[byteutil.ToHex(inst.GetLeft().Bytes())] {
			if _, err := WriteReturn(bs); err != nil {
				return err
			}
		}
	}

	if op == ir.OpSave {
		if _, err := WriteSave(bs, inst.GetLeft().Bytes(), tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpIdent {
		if _, err := WriteIdent(bs, im, inst.GetLeft().Bytes()); err != nil {
			return err
		}
	}

	if op == ir.OpLoad {
		if _, err := WriteLoad(bs, im, inst.GetLeft().Bytes()); err != nil {
			return err
		}
	}

	if op == ir.OpGetFeed {
		if _, err := WriteGetArg(bs, inst.GetLeft().Bytes(), tapeSize); err != nil {
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
		if _, err := WriteMask(bs, tapeSize); err != nil {
			return err
		}
	}

	if op == ir.OpIf {
		// The IR skips ahead when the test is false and the EVM jumps when what it pops is
		// not zero, so the test is turned over first.
		if _, err := bs.Write([]byte{OpIsZero}); err != nil {
			return err
		}
		if _, err := WritePush2(bs, target); err != nil {
			return err
		}
		if _, err := bs.Write([]byte{OpJumpIf}); err != nil {
			return err
		}
	}

	if op == ir.OpJump {
		if _, err := WritePush2(bs, target); err != nil {
			return err
		}
		if _, err := bs.Write([]byte{OpJump}); err != nil {
			return err
		}
	}

	return nil
}

// WriteCode emits a scope: measure, resolve, write.
//
// A jump forward needs the address of code that has not been written, so the bytes are
// measured first and the addresses fall out of the measurement. Every push is a fixed size, so
// one pass is enough — nothing here has to be measured again because an address turned out
// longer than it was guessed.
//
// The base is where this scope lands in the runtime, since a jump carries an address in the
// contract and not an offset into a scope. It is zero while a scope is being measured, and the
// measurement does not depend on it.
func WriteCode(bs io.Writer, im *IdentManager, insts []ir.Instruction, tapeSize int, base int) (int, error) {
	landings := landingsOf(insts)
	arms := armsOf(insts)

	positions, err := PositionsOf(insts, tapeSize, landings, arms)
	if err != nil {
		return 0, err
	}

	for at, inst := range insts {
		if landings[at] {
			if _, err := bs.Write([]byte{OpJumpDestiny}); err != nil {
				return 0, err
			}
		}
		if err := WriteInstruction(bs, im, inst, tapeSize, base+targetOf(inst, at, positions), arms); err != nil {
			return 0, err
		}
	}

	// The way out of an "if" that ends a scope lands past the last instruction.
	if landings[len(insts)] {
		if _, err := bs.Write([]byte{OpJumpDestiny}); err != nil {
			return 0, err
		}
	}

	return bs.Write([]byte{OpStop})
}
