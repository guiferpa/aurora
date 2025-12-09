package byteutil

import "encoding/binary"

func FromUint64(v uint64) []byte {
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, v)
	return r
}

// ToUint64 converts a byte array to uint64 for arithmetic operations.
// It pads the array to 8 bytes (right-aligned) if smaller, or takes the last 8 bytes if larger.
// This ensures arithmetic operations work consistently on 64-bit integers.
func ToUint64(bs []byte) uint64 {
	padded := Padding64Bits(bs)
	// If array is larger than 8 bytes, we only use the first 8 bytes
	// This means tapes larger than 8 bytes will have their extra bytes ignored
	if len(bs) > 8 {
		// Use first 8 bytes (most significant bytes in big-endian)
		return binary.BigEndian.Uint64(padded[:8])
	}
	return binary.BigEndian.Uint64(padded)
}

func FromUint32(v uint32) []byte {
	r := make([]byte, 4)
	binary.BigEndian.PutUint32(r, v)
	return r
}
