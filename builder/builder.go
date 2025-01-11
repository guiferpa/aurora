package builder

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

type Builder interface {
	Build(io.Writer) (int, error)
}

type blr struct {
	insts []emitter.Instruction
}

func getLabelBytes(label []byte) ([]byte, []byte) {
	labellen := make([]byte, 4)
	binary.BigEndian.PutUint32(labellen, uint32(len(label)))
	return labellen, label
}

func getLineBytes(lblen, label, op, left, right []byte) []byte {
	lnlen := make([]byte, 4)
	binary.BigEndian.PutUint32(lnlen, uint32(len(lnlen)+len(label)+len(op)+len(left)+len(right)))
	return bytes.Join([][]byte{lnlen, lblen, label, op, left, right}, []byte(""))
}

func (b *blr) Build(w io.Writer) (int, error) {
	var size int = 0
	var err error
	for _, inst := range b.insts {
		lblen, label := getLabelBytes(inst.GetLabel())  // Len 4 bytes, dynamic size
		op := []byte{inst.GetOpCode()}                  // 1 byte
		left := byteutil.Padding64Bits(inst.GetLeft())  // 8 byte
		right := byteutil.Padding64Bits(inst.GetLeft()) // 8 byte
		line := getLineBytes(lblen, label, op, left, right)
		size += len(line)
		size, err = w.Write(line)
		if err != nil {
			return size, err
		}
	}
	return size, nil
}

func New(insts []emitter.Instruction) *blr {
	return &blr{
		insts: insts,
	}
}
