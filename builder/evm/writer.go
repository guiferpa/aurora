package evm

import (
	"bytes"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

const MEMORY_SLOT_SIZE = 32
const INSTANTIATE_BLOCK_SIZE = 12

func WriteBool(w io.Writer, v byte) (int, error) {
	if _, err := w.Write([]byte{OpPush1, v}); err != nil {
		return 0, err
	}
	return 0, nil
}

func WriteAdd(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpAdd}); err != nil {
		return 0, err
	}
	return 0, nil
}

func WriteMultiply(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMul}); err != nil {
		return 0, err
	}
	return 0, nil
}

func WriteSubtract(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpSub}); err != nil {
		return 0, err
	}
	return 0, nil
}

func WriteDivide(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpDiv}); err != nil {
		return 0, err
	}
	return 0, nil
}

func WriteSave(w io.Writer, left []byte) (int, error) {
	if len(left) == 1 {
		if _, err := WriteBool(w, left[0]); err != nil {
			return 0, err
		}
	}
	if _, err := w.Write([]byte{OpPush8}); err != nil {
		return 0, err
	}
	if _, err := w.Write(left); err != nil {
		return 0, err
	}
	return 0, nil
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

func WriteGetArg(w io.Writer, left []byte) (int, error) {
	index := byteutil.ToUint64(left)
	offset := GetCalldataArgsOffset(index)
	if _, err := w.Write([]byte{OpPush1, offset}); err != nil {
		return 0, err
	}
	return w.Write([]byte{OpCallDataLoad})
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

func WriteCode(bs io.Writer, im *IdentManager, insts []emitter.Instruction) (int, error) {
	for _, inst := range insts {
		op := inst.GetOpCode()

		if op == emitter.OpAdd {
			if _, err := WriteAdd(bs); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpMultiply {
			if _, err := WriteMultiply(bs); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpSubtract {
			if _, err := WriteSubtract(bs); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpDivide {
			if _, err := WriteDivide(bs); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpReturn {
			if _, err := WriteReturn(bs); err != nil {
				return 0, err
			}
		}

		// Push to stack
		if op == emitter.OpSave {
			if _, err := WriteSave(bs, inst.GetLeft()); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpIdent {
			if _, err := WriteIdent(bs, im, inst.GetLeft()); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpLoad {
			if _, err := WriteLoad(bs, im, inst.GetLeft()); err != nil {
				return 0, err
			}
		}

		if op == emitter.OpGetArg {
			if _, err := WriteGetArg(bs, inst.GetLeft()); err != nil {
				return 0, err
			}
		}
	}

	return bs.Write([]byte{OpStop})
}
