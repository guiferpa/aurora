package byteutil

import "encoding/binary"

func FromUint64(v uint64) []byte {
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, v)
	return r
}

func ToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
