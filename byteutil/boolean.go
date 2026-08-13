package byteutil

// ToBoolean reads a tape as a condition: any non-zero byte is true.
//
// There is no boolean representation of its own — true is a tape holding 1, false a tape
// of zeros — so this only asks whether the value is zero. See TrueTape and FalseTape.
func ToBoolean(bs []byte) bool {
	return !IsZeroTape(bs)
}
