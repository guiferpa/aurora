package evm

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

func ToOpByte(op uint32) []byte {
	return byteutil.NoPadding(byteutil.FromUint32(op))
}

type Transformer struct {
	identifiers map[string]byte
}

func (t *Transformer) Transform(w io.Writer, insts []emitter.Instruction) error {
	for _, inst := range insts {
		op := inst.GetOpCode()
		left := inst.GetLeft()
		//_ := inst.GetRight()

		if op == emitter.OpAdd {
			if _, err := w.Write([]byte{OpAdd}); err != nil {
				return err
			}
		}

		if op == emitter.OpMultiply {
			if _, err := w.Write([]byte{OpMul}); err != nil {
				return err
			}
		}

		if op == emitter.OpSubtract {
			if _, err := w.Write([]byte{OpSub}); err != nil {
				return err
			}
		}

		if op == emitter.OpDivide {
			if _, err := w.Write([]byte{OpDiv}); err != nil {
				return err
			}
		}

		if op == emitter.OpSave {
			_, err := w.Write([]byte{OpPush8})
			_, err = w.Write(byteutil.Padding64Bits(left))
			if err != nil {
				return err
			}
		}

		if op == emitter.OpIdent {
			id := fmt.Sprintf("%x", left)
			ref := byte(len(t.identifiers))
			t.identifiers[id] = ref
			_, err := w.Write([]byte{OpPush1})
			_, err = w.Write([]byte{ref * 8})
			_, err = w.Write([]byte{OpMemoryStore8})
			if err != nil {
				return err
			}
		}

		if op == emitter.OpLoad {
			id := fmt.Sprintf("%x", left)
			ref := t.identifiers[id]
			_, err := w.Write([]byte{OpPush1})
			_, err = w.Write([]byte{ref * 8})
			_, err = w.Write([]byte{OpMemoryLoad})
			if err != nil {
				return err
			}
		}

		if op == emitter.OpPrint {
			id := fmt.Sprintf("%x", left)
			ref := t.identifiers[id]
			_, err := w.Write([]byte{OpPush1})
			_, err = w.Write([]byte{ref * 8})
			_, err = w.Write([]byte{OpMemoryStore})
			_, err = w.Write([]byte{OpPush1})
			_, err = w.Write([]byte{0x08}) // 4 bytes
			_, err = w.Write([]byte{OpPush1})
			_, err = w.Write([]byte{ref * 8})
			_, err = w.Write([]byte{0xF3}) // OpReturn
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewTransformer() *Transformer {
	return &Transformer{
		identifiers: make(map[string]byte, 0),
	}
}
