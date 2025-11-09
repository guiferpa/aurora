package byteutil

// ExtractSignificantBytes extracts bytes from the first non-zero byte to the end.
// If the array is all zeros, returns a single zero byte.
// This is useful for operations like pull and push where we need to work with
// only the meaningful bytes of a value.
func ExtractSignificantBytes(bs []byte) []byte {
	if len(bs) == 0 {
		return []byte{0}
	}
	// Find first non-zero byte
	start := 0
	for start < len(bs) && bs[start] == 0 {
		start++
	}
	// If all zeros, return single zero byte
	if start == len(bs) {
		return []byte{0}
	}
	// Return bytes from first non-zero to end
	return bs[start:]
}

