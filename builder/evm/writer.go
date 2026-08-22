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
}

// WriteIdent stores a value under a name, in a slot of memory of its own.
//
// The address goes in two bytes. It used to go in one, and a slot is thirty-two wide, so the
// ninth name in a contract was given the address of the first — 8 * 32 is 256, and one byte
// holds none of it. Two names became one piece of memory, and each wrote over the other.
func WriteIdent(w io.Writer, m IdentOffsetMapper, ident []byte) (int, error) {
	offset := int(m.GetLength()) * MEMORY_SLOT_SIZE
	if _, err := WritePush2(w, offset); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	m.SetOffset(string(ident), offset)
	return 0, nil
}

func WriteLoad(w io.Writer, m IdentOffsetMapper, left []byte) (int, error) {
	if _, err := WritePush2(w, m.GetOffset(left)); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpMemoryLoad})
}

// WriteReturn assumes the return value is on the stack (e.g. after ADD). It stores it at
// mem[0] with MSTORE then returns 32 bytes from 0 so RETURN works without prior memory use.
func WriteReturn(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpPush1, 0x20, OpPush1, 0x00, OpReturn})
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

func WriteDispatchers(bs io.Writer, ds []Dispatcher) (int, error) {
	dispatcherLen := DISPATCHER_BYTES_SIZE * len(ds)
	// After dispatchers we have the no-match dispatcher (STOP); referenced code starts after it.
	referencedStart := dispatcherLen + NO_MATCH_DISPATCHER_SIZE

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
		return dispatcherLen + NO_MATCH_DISPATCHER_SIZE, nil
	}

	return dispatcherLen, nil
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

func WriteCode(bs io.Writer, im *IdentManager, insts []ir.Instruction, tapeSize int) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

		if handled[op] && op != ir.OpSave {
			if err := WriteImmediates(bs, inst, tapeSize); err != nil {
				return 0, err
			}
		}

		if op == ir.OpAdd {
			if _, err := WriteAdd(bs); err != nil {
				return 0, err
			}
			if _, err := WriteMask(bs, tapeSize); err != nil {
				return 0, err
			}
		}

		if op == ir.OpMultiply {
			if _, err := WriteMultiply(bs); err != nil {
				return 0, err
			}
			if _, err := WriteMask(bs, tapeSize); err != nil {
				return 0, err
			}
		}

		if op == ir.OpSubtract {
			if _, err := WriteSubtract(bs); err != nil {
				return 0, err
			}
			if _, err := WriteMask(bs, tapeSize); err != nil {
				return 0, err
			}
		}

		if op == ir.OpDivide {
			if _, err := WriteDivide(bs); err != nil {
				return 0, err
			}
		}

		if op == ir.OpReturn {
			if _, err := WriteReturn(bs); err != nil {
				return 0, err
			}
		}

		if op == ir.OpSave {
			if _, err := WriteSave(bs, inst.GetLeft().Bytes(), tapeSize); err != nil {
				return 0, err
			}
		}

		if op == ir.OpIdent {
			if _, err := WriteIdent(bs, im, inst.GetLeft().Bytes()); err != nil {
				return 0, err
			}
		}

		if op == ir.OpLoad {
			if _, err := WriteLoad(bs, im, inst.GetLeft().Bytes()); err != nil {
				return 0, err
			}
		}

		if op == ir.OpGetFeed {
			if _, err := WriteGetArg(bs, inst.GetLeft().Bytes(), tapeSize); err != nil {
				return 0, err
			}
		}
	}

	return bs.Write([]byte{OpStop})
}
