package byteutil

import "bytes"

// Nothing is the universal neutral value of the language.
// Internal representation: 8 zero bytes. It is not an error, not null, not absence.
// Do not mutate; use a copy if you need to modify.
var Nothing = []byte{0, 0, 0, 0, 0, 0, 0, 0}

// IsNothing reports whether b is the nothing value (8 zero bytes).
func IsNothing(b []byte) bool {
	v := Padding64Bits(b)
	return bytes.Equal(v, Nothing)
}
