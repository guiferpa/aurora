package byteutil

// IsNothing reports whether b is the neutral value: a tape of zeros.
//
// It is the same representation as the number zero, on purpose — "nothing" is a normal
// value meaning neutral, not an absence and not an error. See NothingTape.
func IsNothing(b []byte) bool {
	return len(b) > 0 && IsZeroTape(b)
}
