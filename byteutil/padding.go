package byteutil

func Padding64Bits(bfs []byte) []byte {
	const size = 8
	if len(bfs) >= size {
		return bfs
	}
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
