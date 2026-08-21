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
	GetOffset(ident []byte) byte
	SetOffset(ident string, offset byte)
	GetLength() uint
}

func WriteIdent(w io.Writer, m IdentOffsetMapper, ident []byte) (int, error) {
	// offset fits in byte only if idents count * MEMORY_SLOT_SIZE < 256 (e.g. up to 7 slots of 32).
	offset := byte(m.GetLength() * MEMORY_SLOT_SIZE)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMemoryStore}); err != nil {
		return 0, err
	}
	m.SetOffset(string(ident), offset)
	return 0, nil
}

func WriteLoad(w io.Writer, m IdentOffsetMapper, left []byte) (int, error) {
	offset := m.GetOffset(left)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
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

func WriteInstantiateBlock(w io.Writer, runtimeSize byte) (int, error) {
	if _, err := w.Write([]byte{OpPush1, runtimeSize}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, 0x0c}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := w.Write([]byte{OpCodeCopy}); err != nil { // 1 byte
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, runtimeSize}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := w.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return 0, err
	}
	return w.Write([]byte{OpReturn}) // 1 byte
}

func WriteNoMatchDispatcher(w io.Writer) (int, error) {
	return w.Write([]byte{OpStop})
}

func WriteDispatcher(bs io.Writer, id string, jumpTo int) (int, error) {
	if _, err := bs.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := bs.Write([]byte{OpCallDataLoad}); err != nil { // 1 byte
		return 0, err
	}
	// Isolate the first 4 bytes of the keccak256 hash of the id
	if _, err := bs.Write([]byte{OpPush1, byte((CALLDATA_SLOT_READABLE - 4) * BYTE_SIZE)}); err != nil { // 2 bytes
		return 0, err
	}
	if _, err := bs.Write([]byte{OpShiftRight}); err != nil { // 1 byte
		return 0, err
	}
	selector := crypto.Keccak256([]byte(id))[:4]
	if _, err := bs.Write(append([]byte{OpPush4}, selector...)); err != nil { // 5 bytes
		return 0, err
	}
	if _, err := bs.Write([]byte{OpEqual}); err != nil { // 1 byte
		return 0, err
	}
	// PUSH1 limits jumpTo to 0–255; larger runtimes would need PUSH2.
	if _, err := bs.Write([]byte{OpPush1, byte(jumpTo)}); err != nil { // 2 bytes
		return 0, err
	}
	return bs.Write([]byte{OpJumpIf}) // 1 byte
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

func WriteCode(bs io.Writer, im *IdentManager, insts []ir.Instruction, tapeSize int) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

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
