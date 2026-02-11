package byteutil

// Padding64Bits ensures a byte array is exactly 8 bytes with right-aligned padding.
// If larger than 8 bytes, takes only the last 8 bytes (right-aligned).
// If smaller than 8 bytes, pads with zeros on the left (right-aligned).
func Padding64Bits(bfs []byte) []byte {
	const size = 8
	if len(bfs) > size {
		// If larger than 8 bytes, take only the last 8 bytes (right-aligned)
		return bfs[len(bfs)-size:]
	}
	if len(bfs) == size {
		return bfs
	}
	// Pad with zeros on the left (right-aligned)
	bs := make([]byte, size)
	for i := 0; i < len(bfs); i++ {
		bs[(size-len(bfs))+i] = bfs[i]
	}
	return bs
}

func Padding32Bits(bfs []byte) []byte {
	const size = 4
	if len(bfs) >= size {
		return bfs
	}
	bs := make([]byte, size)
	for i := 0; i < len(bfs); i++ {
		bs[(size-len(bfs))+i] = bfs[i]
	}
	return bs
}

func NoPadding(bfs []byte) []byte {
	var i int
	for i < len(bfs) {
		if bfs[i] == 0b0 {
			i++
			continue
		}
		return bfs[i:]
	}
	return bfs
}

func Padding32Bytes(bfs []byte) []byte {
	const size = 32
	bs := make([]byte, size)
	for i := 0; i < len(bfs); i++ {
		bs[(size-len(bfs))+i] = bfs[i]
	}
	return bs
}
