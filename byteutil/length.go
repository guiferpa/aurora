package byteutil

func NonZeroFilledLength(v []byte) int {
	i := 0
	for range v {
		if v[i] == 0b0 {
			i++
		}
	}
	return len(v) - i
}
