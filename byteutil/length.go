package byteutil

const MAX_BYTES_VARIANTS = 1<<8
const MAX_BYTES = MAX_BYTES_VARIANTS - 1

func NonZeroFilledLength(v []byte) int {
	i := len(v)
	c := 0
	ec := false
	for i > 0 {
		if v[i-1] != 0b0 {
			ec = true
		}
		if ec && v[i-1] == 0b0 {
			c++
			ec = false
		}
		i--
	}
	return c
}
