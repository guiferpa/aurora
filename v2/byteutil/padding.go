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
