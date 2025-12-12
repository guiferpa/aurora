package evm

import (
	"bytes"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
)

func writeDispatcher(w io.Writer, id string) (int, error) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := bs.Write([]byte{OpPush1}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{0x00}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpCallDataLoad}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpPush1}); err != nil {
		return 0, err
	}
	// Isolate the first 4 bytes of the keccak256 hash of the id
	if _, err := bs.Write([]byte{byte((CALLDATA_SLOT_READABLE - 4) * BYTE_SIZE)}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpShiftRight}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpPush4}); err != nil {
		return 0, err
	}
	selector := crypto.Keccak256([]byte(id))[:4]
	if _, err := bs.Write(selector); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpEqual}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpPush1}); err != nil {
		return 0, err
	}
	offset := bs.Len() + 3
	if _, err := bs.Write([]byte{byte(offset)}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpJumpIf}); err != nil {
		return 0, err
	}
	if _, err := bs.Write([]byte{OpStop}); err != nil {
		return 0, err
	}
	if _, err := io.Copy(w, bs); err != nil {
		return 0, err
	}
	return bs.Len(), nil
}
