package token

import "bytes"

// Equal compares two tokens by what they matched and what kind of thing they are, which is
// what a test asks about. Where they sit in the source is deliberately not part of it: the
// same word written on another line is the same token.
func Equal(a, b Token) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return bytes.Equal(a.GetMatch(), b.GetMatch()) && a.GetTag() == b.GetTag()
}
