package byteutil

import "bytes"

// Nothing is the universal neutral value of the Aurora language.
// Internal representation: 8 zero bytes. It is not an error, not null, not absence.
// Do not mutate; use a copy if you need to modify.
// See docs/language-design.md for the language-level description of nothing.
var Nothing = []byte{}

// IsNothing reports whether b is the nothing value (8 zero bytes).
func IsNothing(b []byte) bool {
	return b != nil && bytes.Equal(b, Nothing)
}
