package evm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

const (
	DISPATCHER_BYTES_SIZE         = 15
	DISPATCHER_SELECTOR_NAME_SIZE = 4
	// NO_MATCH_DISPATCHER_SIZE is the bytecode size for "selector not found": just STOP (1 byte).
	NO_MATCH_DISPATCHER_SIZE = 1
	CALLDATA_SLOT_READABLE   = 32
)

func GetCalldataArgsOffset(index uint64) byte {
	return CALLDATA_SLOT_READABLE << index
}

type RuntimeCodeReferenced struct {
	Selector []byte
	Offset   int
	Length   int
	Code     *bytes.Buffer
}

func (t *Builder) buildCode(insts []emitter.Instruction) (*bytes.Buffer, error) {
	bs := bytes.NewBuffer(make([]byte, 0))

	for _, inst := range insts {
		op := inst.GetOpCode()

		if op == emitter.OpAdd {
			if _, err := t.writeAdd(bs); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpMultiply {
			if _, err := t.writeMult(bs); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpSubtract {
			if _, err := t.writeSub(bs); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpDivide {
			if _, err := t.writeDiv(bs); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpResult {
			if _, err := bs.Write([]byte{OpPush1, 0x00, OpMemoryStore}); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpReturn {
			if _, err := bs.Write([]byte{OpPush1, 0x20, OpPush1, 0x00, OpReturn}); err != nil {
				return nil, err
			}
		}

		if op == emitter.OpSave {
			if left := inst.GetLeft(); len(left) == 1 {
				if _, err := t.writeBool(bs, left[0]); err != nil {
					return nil, err
				}
			}
			t.operands = append(t.operands, inst.GetLeft())
		}

		if op == emitter.OpGetArg {
			// arguments N -> load from calldata at offset 4 + N*32 (ABI: selector then 32-byte slots)
			index := byteutil.ToUint64(inst.GetLeft())
			offset := GetCalldataArgsOffset(index)
			if _, err := bs.Write([]byte{OpPush1, offset}); err != nil {
				return nil, err
			}
			if _, err := bs.Write([]byte{OpCallDataLoad}); err != nil {
				return nil, err
			}
			// Value is on stack; add/sub/etc use writePush8SafeFromOperands which is no-op when operands empty, so the next op will use this value.
		}
	}

	if _, err := bs.Write([]byte{OpStop}); err != nil {
		return nil, err
	}

	return bs, nil
}

type RuntimeCode struct {
	Root       *bytes.Buffer
	Referenced []RuntimeCodeReferenced
}

func (t *Builder) getInstructions() []emitter.Instruction {
	instructions := make([]emitter.Instruction, 0)
	for t.cursor < len(t.insts) {
		t.cursor++
		inst := t.insts[t.cursor]
		instructions = append(instructions, inst)
		if inst.GetOpCode() == emitter.OpReturn {
			break
		}
	}
	return instructions
}

func (t *Builder) genSelector() []byte {
	if len(t.insts) <= t.cursor+2 {
		return nil
	}
	inst := t.insts[t.cursor+2]
	if inst.GetOpCode() != emitter.OpIdent {
		return nil
	}
	return inst.GetLeft()

}

func (t *Builder) pickRuntimeCode() (*RuntimeCode, error) {
	referenced := make([]RuntimeCodeReferenced, 0)
	rootinsts := make([]emitter.Instruction, 0)
	offset := 0

	for t.cursor < len(t.insts) {
		inst := t.insts[t.cursor]
		if inst.GetOpCode() == emitter.OpBeginScope {
			code, err := t.buildCode(t.getInstructions())
			if err != nil {
				return nil, err
			}
			selector := t.genSelector()
			if selector == nil {
				t.cursor++
				continue
			}
			referenced = append(referenced, RuntimeCodeReferenced{
				Selector: selector,
				// MUST add OpJumpDestiny to the code to jump to the dispatcher
				Code:   bytes.NewBuffer(append([]byte{OpJumpDestiny}, code.Bytes()...)),
				Offset: offset,
				Length: code.Len(),
			})
			// +1: each block in "referenced" is prefixed by OpJumpDestiny (1 byte)
			offset += 1 + code.Len()
		} else {
			rootinsts = append(rootinsts, inst)
		}
		t.cursor++
	}

	if len(rootinsts) > 0 {
		root, err := t.buildCode(rootinsts)
		if err != nil {
			return nil, err
		}
		return &RuntimeCode{Root: root, Referenced: referenced}, nil
	}

	return &RuntimeCode{Referenced: referenced}, nil
}

func buildDispatcher(id string, jumpTo int) (*bytes.Buffer, error) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := bs.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := bs.Write([]byte{OpCallDataLoad}); err != nil { // 1 byte
		return nil, err
	}
	// Isolate the first 4 bytes of the keccak256 hash of the id
	if _, err := bs.Write([]byte{OpPush1, byte((CALLDATA_SLOT_READABLE - 4) * BYTE_SIZE)}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := bs.Write([]byte{OpShiftRight}); err != nil { // 1 byte
		return nil, err
	}
	selector := crypto.Keccak256([]byte(id))[:4]
	if _, err := bs.Write(append([]byte{OpPush4}, selector...)); err != nil { // 5 bytes
		return nil, err
	}
	if _, err := bs.Write([]byte{OpEqual}); err != nil { // 1 byte
		return nil, err
	}
	if _, err := bs.Write([]byte{OpPush1, byte(jumpTo)}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := bs.Write([]byte{OpJumpIf}); err != nil { // 1 byte
		return nil, err
	}
	return bs, nil
}

// buildNoMatchDispatcher returns bytecode for when no dispatcher matched (invalid selector): just STOP.
func buildNoMatchDispatcher() []byte {
	return []byte{OpStop}
}

func buildDispatchers(rfs []RuntimeCodeReferenced) (*bytes.Buffer, error) {
	bs := bytes.NewBuffer(make([]byte, 0))

	dispatcherLen := DISPATCHER_BYTES_SIZE * len(rfs)
	// After dispatchers we have the no-match dispatcher (STOP); referenced code starts after it.
	referencedStart := dispatcherLen + NO_MATCH_DISPATCHER_SIZE

	for _, rf := range rfs {
		jumpTo := referencedStart + rf.Offset
		dispatcher, err := buildDispatcher(string(rf.Selector), jumpTo)
		if err != nil {
			return nil, err
		}
		if _, err := bs.Write(dispatcher.Bytes()); err != nil {
			return nil, err
		}
	}
	return bs, nil
}

func (t *Builder) buildRuntimeCode() (*bytes.Buffer, error) {
	built := make([]byte, 0)

	rc, err := t.pickRuntimeCode()
	if err != nil {
		return nil, err
	}

	dispatchers, err := buildDispatchers(rc.Referenced)
	if err != nil {
		return nil, err
	}
	built = append(built, dispatchers.Bytes()...)
	built = append(built, buildNoMatchDispatcher()...)

	referenced := make([]byte, 0)
	for _, rf := range rc.Referenced {
		referenced = append(referenced, rf.Code.Bytes()...)
	}
	built = append(built, referenced...)
	if rc.Root != nil {
		built = append(built, rc.Root.Bytes()...)
	}

	return bytes.NewBuffer(built), nil
}
